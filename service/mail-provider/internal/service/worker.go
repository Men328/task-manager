package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type notificationWorker struct {
	subs     SubscriptionRepository
	gmail    GmailClient
	tokens   *TokenManager
	analyzer MailAnalyzer
	tasks    TaskCreator
	queue    *Queue
	opts     Options
}

func NewWorker(
	subs SubscriptionRepository,
	gmail GmailClient,
	tokens *TokenManager,
	analyzer MailAnalyzer,
	tasks TaskCreator,
	queue *Queue,
	opts Options,
) Worker {
	return &notificationWorker{
		subs:     subs,
		gmail:    gmail,
		tokens:   tokens,
		analyzer: analyzer,
		tasks:    tasks,
		queue:    queue,
		opts:     opts.withDefaults(),
	}
}

func (w *notificationWorker) Run(ctx context.Context) error {
	for i := 0; i < w.opts.WorkerCount; i++ {
		go w.consume(ctx)
	}
	go w.renewLoop(ctx)
	<-ctx.Done()
	return nil
}

func (w *notificationWorker) consume(ctx context.Context) {
	for {
		notification, ok := w.queue.dequeue(ctx)
		if !ok {
			return
		}
		w.process(ctx, notification)
	}
}

func (w *notificationWorker) process(ctx context.Context, notification model.Notification) {
	sub, err := w.subs.GetByProfileID(ctx, notification.ProfileID)
	if err != nil {
		slog.Error("không tìm thấy đăng ký cho notice", "profile_id", notification.ProfileID, "error", err)
		return
	}

	accessToken, err := w.tokens.AccessToken(ctx, sub)
	if err != nil {
		slog.Error("không lấy được access token", "profile_id", sub.ProfileID, "error", err)
		return
	}

	messageIDs, latestHistoryID, err := w.gmail.NewMessageIDs(ctx, accessToken, sub.HistoryID, w.opts.LabelIDs)
	if errors.Is(err, model.ErrHistoryGone) {
		w.resetCheckpoint(ctx, sub, accessToken)
		return
	}
	if err != nil {
		slog.Error("đọc lịch sử gmail thất bại", "profile_id", sub.ProfileID, "email", sub.Email, "error", err)
		return
	}

	if w.opts.AnalyzerEnabled {
		for _, messageID := range messageIDs {
			w.processMessage(ctx, sub, accessToken, messageID)
		}
	}

	checkpoint := latestHistoryID
	if checkpoint == "" {
		checkpoint = notification.HistoryID
	}
	if checkpoint != "" && checkpoint != sub.HistoryID {
		if _, err := w.subs.UpdateWatch(ctx, sub.ProfileID, checkpoint, sub.WatchExpiresAt); err != nil {
			slog.Error("cập nhật checkpoint thất bại", "profile_id", sub.ProfileID, "error", err)
		}
	}
}

func (w *notificationWorker) processMessage(ctx context.Context, sub model.Subscription, accessToken string, messageID string) {
	message, err := w.gmail.Message(ctx, accessToken, messageID)
	if err != nil {
		slog.Error("đọc email gốc thất bại", "profile_id", sub.ProfileID, "message_id", messageID, "error", err)
		return
	}
	if strings.TrimSpace(message.Subject) == "" && strings.TrimSpace(message.Body) == "" {
		return
	}

	draft, err := w.analyzer.Analyze(ctx, message)
	if err != nil {
		slog.Error("phân tích email thất bại", "profile_id", sub.ProfileID, "message_id", messageID, "error", err)
		return
	}
	if !draft.Actionable || strings.TrimSpace(draft.Title) == "" {
		return
	}
	if draft.Priority == model.PriorityUnspecified {
		draft.Priority = w.opts.DefaultPriority
	}

	created, err := w.tasks.Create(ctx, model.TaskInput{
		ProfileID:   sub.ProfileID,
		Title:       draft.Title,
		Description: draft.Description,
		Priority:    draft.Priority,
		DueAt:       draft.DueAt,
		Source:      message.ID,
	})
	if err != nil {
		slog.Error("tạo task từ email thất bại", "profile_id", sub.ProfileID, "message_id", messageID, "error", err)
		return
	}

	slog.Info("đã tạo task từ email",
		"profile_id", sub.ProfileID,
		"email", sub.Email,
		"message_id", messageID,
		"task_id", created.ID,
	)
}

func (w *notificationWorker) resetCheckpoint(ctx context.Context, sub model.Subscription, accessToken string) {
	current, err := w.gmail.ProfileHistoryID(ctx, accessToken)
	if err != nil {
		slog.Error("đọc profile gmail thất bại", "profile_id", sub.ProfileID, "error", err)
		return
	}
	if _, err := w.subs.UpdateWatch(ctx, sub.ProfileID, current, sub.WatchExpiresAt); err != nil {
		slog.Error("đặt lại checkpoint thất bại", "profile_id", sub.ProfileID, "error", err)
		return
	}
	slog.Warn("checkpoint historyId quá cũ, đã bỏ qua các email trong khoảng bị mất",
		"profile_id", sub.ProfileID,
		"email", sub.Email,
		"history_id", current,
	)
}

func (w *notificationWorker) renewLoop(ctx context.Context) {
	if w.opts.RenewInterval <= 0 || strings.TrimSpace(w.opts.TopicName) == "" {
		return
	}
	ticker := time.NewTicker(w.opts.RenewInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.renewWatches(ctx)
		}
	}
}

func (w *notificationWorker) renewWatches(ctx context.Context) {
	subs, err := w.subs.List(ctx)
	if err != nil {
		slog.Error("đọc danh sách đăng ký thất bại", "error", err)
		return
	}

	threshold := time.Now().Add(w.opts.RenewThreshold)
	for _, sub := range subs {
		if sub.WatchExpiresAt.After(threshold) {
			continue
		}
		accessToken, err := w.tokens.AccessToken(ctx, sub)
		if err != nil {
			slog.Error("không lấy được access token để gia hạn", "profile_id", sub.ProfileID, "error", err)
			continue
		}
		watch, err := w.gmail.Watch(ctx, accessToken, w.opts.TopicName, w.opts.LabelIDs)
		if err != nil {
			slog.Error("gia hạn watch thất bại", "profile_id", sub.ProfileID, "error", err)
			continue
		}
		historyID := sub.HistoryID
		if historyID == "" {
			historyID = watch.HistoryID
		}
		if _, err := w.subs.UpdateWatch(ctx, sub.ProfileID, historyID, watch.ExpiresAt); err != nil {
			slog.Error("lưu watch mới thất bại", "profile_id", sub.ProfileID, "error", err)
			continue
		}
		slog.Info("đã gia hạn watch gmail", "profile_id", sub.ProfileID, "watch_expires_at", watch.ExpiresAt)
	}
}
