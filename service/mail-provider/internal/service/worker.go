package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path"
	"strings"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type archivedEmail struct {
	ID          string               `json:"id"`
	ThreadID    string               `json:"thread_id"`
	From        string               `json:"from"`
	To          string               `json:"to"`
	Subject     string               `json:"subject"`
	Snippet     string               `json:"snippet"`
	Body        string               `json:"body"`
	ReceivedAt  time.Time            `json:"received_at"`
	Attachments []archivedAttachment `json:"attachments"`
}

type archivedAttachment struct {
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

type notificationWorker struct {
	subs      SubscriptionRepository
	gmail     GmailClient
	tokens    *TokenManager
	analyzer  MailAnalyzer
	tasks     TaskCreator
	schedules ScheduleCreator
	events    EventCreator
	backlogs  BacklogCreator
	storage   BlobStore
	queue     *Queue
	opts      Options
}

func NewWorker(
	subs SubscriptionRepository,
	gmail GmailClient,
	tokens *TokenManager,
	analyzer MailAnalyzer,
	tasks TaskCreator,
	schedules ScheduleCreator,
	events EventCreator,
	backlogs BacklogCreator,
	storage BlobStore,
	queue *Queue,
	opts Options,
) Worker {
	return &notificationWorker{
		subs:      subs,
		gmail:     gmail,
		tokens:    tokens,
		analyzer:  analyzer,
		tasks:     tasks,
		schedules: schedules,
		events:    events,
		backlogs:  backlogs,
		storage:   storage,
		queue:     queue,
		opts:      opts.withDefaults(),
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

	if w.opts.AnalyzerEnabled || w.opts.RuleFallbackEnabled {
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

	objectKey := w.archive(ctx, sub, accessToken, message)

	draft := w.classify(ctx, message)
	if draft.Priority == model.PriorityUnspecified {
		draft.Priority = w.opts.DefaultPriority
	}

	w.dispatch(ctx, sub, message, draft, objectKey)
}

func (w *notificationWorker) classify(ctx context.Context, message model.EmailMessage) model.MailDraft {
	if w.opts.AnalyzerEnabled {
		draft, err := w.analyzer.Analyze(ctx, message)
		if err == nil && draft.Category.Valid() {
			return draft
		}
		slog.Warn("DeepSeek phân loại thất bại, dùng rule keyword dự phòng",
			"message_id", message.ID,
			"error", err,
		)
	}
	if w.opts.RuleFallbackEnabled {
		return classifyByRules(message)
	}
	return model.MailDraft{Category: model.CategoryOther, Reason: "classifier_disabled"}
}

func (w *notificationWorker) dispatch(ctx context.Context, sub model.Subscription, message model.EmailMessage, draft model.MailDraft, objectKey string) {
	switch draft.Category {
	case model.CategoryTask:
		if !draft.Actionable || strings.TrimSpace(draft.Title) == "" {
			w.saveBacklog(ctx, sub, message, draft, objectKey, "not_actionable")
			return
		}
		w.createTask(ctx, sub, message, draft)
	case model.CategorySchedule:
		start := firstTime(draft.StartAt, draft.DueAt)
		if strings.TrimSpace(draft.Title) == "" || start == nil {
			w.saveBacklog(ctx, sub, message, draft, objectKey, "missing_schedule_time")
			return
		}
		w.createSchedule(ctx, sub, message, draft, *start)
	case model.CategoryEvent:
		start := firstTime(draft.StartAt, draft.DueAt)
		if strings.TrimSpace(draft.Title) == "" || start == nil {
			w.saveBacklog(ctx, sub, message, draft, objectKey, "missing_event_time")
			return
		}
		w.createEvent(ctx, sub, message, draft, *start)
	default:
		w.saveBacklog(ctx, sub, message, draft, objectKey, draftReason(draft))
	}
}

func (w *notificationWorker) createTask(ctx context.Context, sub model.Subscription, message model.EmailMessage, draft model.MailDraft) {
	created, err := w.tasks.Create(ctx, model.TaskInput{
		ProfileID:   sub.ProfileID,
		Title:       draft.Title,
		Description: draft.Description,
		Priority:    draft.Priority,
		DueAt:       draft.DueAt,
		Source:      message.ID,
	})
	if err != nil {
		slog.Error("tạo task từ email thất bại", "profile_id", sub.ProfileID, "message_id", message.ID, "error", err)
		return
	}
	slog.Info("đã tạo task từ email", "profile_id", sub.ProfileID, "message_id", message.ID, "task_id", created.ID)
}

func (w *notificationWorker) createSchedule(ctx context.Context, sub model.Subscription, message model.EmailMessage, draft model.MailDraft, start time.Time) {
	created, err := w.schedules.Create(ctx, model.ScheduleInput{
		ProfileID:   sub.ProfileID,
		Title:       draft.Title,
		Description: draft.Description,
		Location:    draft.Location,
		StartAt:     start,
		EndAt:       draft.EndAt,
		AllDay:      draft.AllDay,
	})
	if err != nil {
		slog.Error("tạo lịch từ email thất bại", "profile_id", sub.ProfileID, "message_id", message.ID, "error", err)
		return
	}
	slog.Info("đã tạo lịch từ email", "profile_id", sub.ProfileID, "message_id", message.ID, "schedule_id", created.ID)
}

func (w *notificationWorker) createEvent(ctx context.Context, sub model.Subscription, message model.EmailMessage, draft model.MailDraft, start time.Time) {
	created, err := w.events.Create(ctx, model.EventInput{
		ProfileID:   sub.ProfileID,
		Title:       draft.Title,
		Description: draft.Description,
		Location:    draft.Location,
		StartAt:     start,
		EndAt:       draft.EndAt,
		AllDay:      draft.AllDay,
		Source:      message.ID,
	})
	if err != nil {
		slog.Error("tạo sự kiện từ email thất bại", "profile_id", sub.ProfileID, "message_id", message.ID, "error", err)
		return
	}
	slog.Info("đã tạo sự kiện từ email", "profile_id", sub.ProfileID, "message_id", message.ID, "event_id", created.ID)
}

func (w *notificationWorker) saveBacklog(ctx context.Context, sub model.Subscription, message model.EmailMessage, draft model.MailDraft, objectKey string, reason string) {
	created, err := w.backlogs.Create(ctx, model.BacklogInput{
		ProfileID:   sub.ProfileID,
		Title:       draft.Title,
		Description: draft.Description,
		Sender:      message.From,
		Source:      message.ID,
		Category:    string(draft.Category),
		Reason:      reason,
		ObjectKey:   objectKey,
	})
	if err != nil {
		slog.Error("lưu backlog từ email thất bại", "profile_id", sub.ProfileID, "message_id", message.ID, "error", err)
		return
	}
	slog.Info("đã đẩy email vào backlog",
		"profile_id", sub.ProfileID,
		"message_id", message.ID,
		"category", draft.Category,
		"reason", reason,
		"backlog_id", created.ID,
	)
}

func (w *notificationWorker) archive(ctx context.Context, sub model.Subscription, accessToken string, message model.EmailMessage) string {
	if !w.opts.ArchiveEnabled {
		return ""
	}

	payload := archivedEmail{
		ID:          message.ID,
		ThreadID:    message.ThreadID,
		From:        message.From,
		To:          message.To,
		Subject:     message.Subject,
		Snippet:     message.Snippet,
		Body:        message.Body,
		ReceivedAt:  message.ReceivedAt,
		Attachments: make([]archivedAttachment, 0, len(message.Attachments)),
	}
	for _, attachment := range message.Attachments {
		payload.Attachments = append(payload.Attachments, archivedAttachment{
			Filename: attachment.Filename,
			MimeType: attachment.MimeType,
			Size:     attachment.Size,
		})
	}

	encoded, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		slog.Error("mã hoá email gốc thất bại", "message_id", message.ID, "error", err)
		return ""
	}

	key := archiveKey(sub.ProfileID, message.ID, "")
	stored, err := w.storage.Put(ctx, key, "application/json", encoded)
	if err != nil {
		slog.Warn("lưu email gốc lên storage thất bại", "message_id", message.ID, "error", err)
		return ""
	}

	w.archiveAttachments(ctx, sub, accessToken, message)
	return stored
}

func (w *notificationWorker) archiveAttachments(ctx context.Context, sub model.Subscription, accessToken string, message model.EmailMessage) {
	limit := w.opts.MaxAttachments
	if limit <= 0 || len(message.Attachments) == 0 {
		return
	}

	stored := 0
	for _, attachment := range message.Attachments {
		if stored >= limit {
			slog.Warn("bỏ qua attachment vượt giới hạn", "message_id", message.ID, "limit", limit)
			return
		}
		if attachment.AttachmentID == "" || attachment.Size > w.opts.MaxAttachmentBytes {
			continue
		}

		data, err := w.gmail.Attachment(ctx, accessToken, message.ID, attachment.AttachmentID)
		if err != nil {
			slog.Warn("tải attachment thất bại", "message_id", message.ID, "filename", attachment.Filename, "error", err)
			continue
		}
		if int64(len(data)) > w.opts.MaxAttachmentBytes {
			continue
		}

		key := archiveKey(sub.ProfileID, message.ID, attachment.Filename)
		contentType := attachment.MimeType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if _, err := w.storage.Put(ctx, key, contentType, data); err != nil {
			slog.Warn("lưu attachment lên storage thất bại", "message_id", message.ID, "filename", attachment.Filename, "error", err)
			continue
		}
		stored++
	}
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

func firstTime(values ...*time.Time) *time.Time {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}

func draftReason(draft model.MailDraft) string {
	if strings.TrimSpace(draft.Reason) != "" {
		return draft.Reason
	}
	return "unclassified"
}

func archiveKey(profileID string, messageID string, filename string) string {
	day := time.Now().UTC().Format("20060102")
	base := path.Join("mail", safeSegment(profileID), day, safeSegment(messageID))
	if strings.TrimSpace(filename) == "" {
		return base + ".json"
	}
	return path.Join(base, "attachments", safeSegment(filename))
}

func safeSegment(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_", " ", "_")
	return replacer.Replace(trimmed)
}
