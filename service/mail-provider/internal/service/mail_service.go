package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type mailService struct {
	subs   SubscriptionRepository
	gmail  GmailClient
	tokens *TokenManager
	queue  *Queue
	opts   Options
	now    func() time.Time
}

func NewMailService(
	subs SubscriptionRepository,
	gmail GmailClient,
	tokens *TokenManager,
	queue *Queue,
	opts Options,
) MailService {
	return &mailService{subs: subs, gmail: gmail, tokens: tokens, queue: queue, opts: opts.withDefaults(), now: time.Now}
}

func (s *mailService) Subscribe(ctx context.Context, in model.SubscribeInput) (model.Subscription, error) {
	if strings.TrimSpace(s.opts.TopicName) == "" {
		return model.Subscription{}, model.ErrNotConfigured
	}

	email := strings.ToLower(strings.TrimSpace(in.Email))
	watch, err := s.gmail.Watch(ctx, in.AccessToken, s.opts.TopicName, s.opts.LabelIDs)
	if err != nil {
		slog.Error("gọi users.watch thất bại",
			"profile_id", in.ProfileID,
			"email", email,
			"topic", s.opts.TopicName,
			"label_ids", s.opts.LabelIDs,
			"error", err,
		)
		return model.Subscription{}, err
	}

	now := s.now().UTC()
	sub, err := s.subs.Upsert(ctx, model.Subscription{
		ProfileID:            in.ProfileID,
		Email:                email,
		AccessToken:          in.AccessToken,
		RefreshToken:         in.RefreshToken,
		AccessTokenExpiresAt: in.AccessTokenExpiresAt,
		HistoryID:            watch.HistoryID,
		WatchExpiresAt:       watch.ExpiresAt,
		CreatedAt:            now,
		UpdatedAt:            now,
	})
	if err != nil {
		return model.Subscription{}, err
	}

	slog.Info("đăng ký nhận thông báo gmail",
		"profile_id", sub.ProfileID,
		"email", sub.Email,
		"history_id", sub.HistoryID,
		"watch_expires_at", sub.WatchExpiresAt,
	)
	return sub, nil
}

func (s *mailService) Unsubscribe(ctx context.Context, profileID string) error {
	sub, err := s.subs.GetByProfileID(ctx, profileID)
	if err != nil {
		return err
	}

	accessToken, tokenErr := s.tokens.AccessToken(ctx, sub)
	if tokenErr != nil {
		slog.Warn("không lấy được access token để dừng watch", "profile_id", profileID, "error", tokenErr)
	} else if stopErr := s.gmail.Stop(ctx, accessToken); stopErr != nil {
		slog.Warn("dừng watch gmail thất bại", "profile_id", profileID, "error", stopErr)
	}

	return s.subs.Delete(ctx, profileID)
}

func (s *mailService) GetSubscription(ctx context.Context, profileID string) (model.Subscription, error) {
	return s.subs.GetByProfileID(ctx, profileID)
}

func (s *mailService) HandleNotification(ctx context.Context, email string, historyID string, messageID string) (bool, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" || strings.TrimSpace(historyID) == "" {
		return false, model.ErrInvalidPayload
	}

	sub, err := s.subs.GetByEmail(ctx, normalized)
	if errors.Is(err, model.ErrNotFound) {
		slog.Warn("bỏ qua notice của hộp thư chưa đăng ký", "email", normalized)
		return false, nil
	}
	if err != nil {
		return false, err
	}

	accepted := s.queue.Enqueue(model.Notification{
		ProfileID:  sub.ProfileID,
		Email:      sub.Email,
		HistoryID:  strings.TrimSpace(historyID),
		MessageID:  strings.TrimSpace(messageID),
		ReceivedAt: s.now().UTC(),
	})
	if !accepted {
		return false, model.ErrQueueFull
	}
	return true, nil
}
