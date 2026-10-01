package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

func testOptions() Options {
	return Options{
		TopicName:       "projects/demo/topics/gmail",
		LabelIDs:        []string{"INBOX"},
		WorkerCount:     1,
		DefaultPriority: model.PriorityMedium,
		AnalyzerEnabled: true,
	}
}

func TestSubscribeFailsWhenTopicMissing(t *testing.T) {
	subs := newStubSubscriptions()
	mail := NewMailService(subs, &stubGmail{}, NewTokenManager(subs, &stubRefresher{}), NewQueue(4), Options{})

	_, err := mail.Subscribe(context.Background(), model.SubscribeInput{
		ProfileID:   "p1",
		Email:       "user@example.com",
		AccessToken: "access",
	})
	if !errors.Is(err, model.ErrNotConfigured) {
		t.Fatalf("thiếu topic phải trả ErrNotConfigured, nhận %v", err)
	}
}

func TestSubscribeFailsWhenWatchRejected(t *testing.T) {
	subs := newStubSubscriptions()
	gmail := &stubGmail{watchErr: model.ErrGmailFailed}
	mail := NewMailService(subs, gmail, NewTokenManager(subs, &stubRefresher{}), NewQueue(4), testOptions())

	if _, err := mail.Subscribe(context.Background(), model.SubscribeInput{
		ProfileID:   "p1",
		Email:       "user@example.com",
		AccessToken: "access",
	}); !errors.Is(err, model.ErrGmailFailed) {
		t.Fatalf("watch lỗi phải trả nguyên nhân gốc, nhận %v", err)
	}
}

func TestSubscribeStoresWatchCheckpoint(t *testing.T) {
	subs := newStubSubscriptions()
	expiresAt := time.Now().Add(6 * 24 * time.Hour).UTC()
	gmail := &stubGmail{watchResult: model.WatchResult{HistoryID: "500", ExpiresAt: expiresAt}}
	mail := NewMailService(subs, gmail, NewTokenManager(subs, &stubRefresher{}), NewQueue(4), testOptions())

	subscription, err := mail.Subscribe(context.Background(), model.SubscribeInput{
		ProfileID:            "p1",
		Email:                "User@Example.com",
		AccessToken:          "access",
		RefreshToken:         "refresh",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
	})
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	if subscription.Email != "user@example.com" {
		t.Fatalf("email phải được chuẩn hoá, nhận %q", subscription.Email)
	}
	if subscription.HistoryID != "500" {
		t.Fatalf("historyId phải lưu từ watch, nhận %q", subscription.HistoryID)
	}
	if !subscription.WatchExpiresAt.Equal(expiresAt) {
		t.Fatalf("watch_expires_at sai: %s", subscription.WatchExpiresAt)
	}
	if subscription.RefreshToken != "refresh" {
		t.Fatalf("refresh token phải được lưu, nhận %q", subscription.RefreshToken)
	}
}

func TestHandleNotificationRejectsInvalidPayload(t *testing.T) {
	subs := newStubSubscriptions()
	mail := NewMailService(subs, &stubGmail{}, NewTokenManager(subs, &stubRefresher{}), NewQueue(4), testOptions())

	if _, err := mail.HandleNotification(context.Background(), "", "1", "m"); !errors.Is(err, model.ErrInvalidPayload) {
		t.Fatalf("thiếu email phải là ErrInvalidPayload, nhận %v", err)
	}
	if _, err := mail.HandleNotification(context.Background(), "user@example.com", "", "m"); !errors.Is(err, model.ErrInvalidPayload) {
		t.Fatalf("thiếu historyId phải là ErrInvalidPayload, nhận %v", err)
	}
}

func TestHandleNotificationIgnoresUnknownMailbox(t *testing.T) {
	subs := newStubSubscriptions()
	mail := NewMailService(subs, &stubGmail{}, NewTokenManager(subs, &stubRefresher{}), NewQueue(4), testOptions())

	accepted, err := mail.HandleNotification(context.Background(), "nobody@example.com", "1", "m")
	if err != nil {
		t.Fatalf("hộp thư lạ không được trả lỗi: %v", err)
	}
	if accepted {
		t.Fatal("hộp thư lạ phải trả accepted=false")
	}
}

func TestHandleNotificationFailsWhenQueueFull(t *testing.T) {
	subs := newStubSubscriptions(model.Subscription{ProfileID: "p1", Email: "user@example.com"})
	mail := NewMailService(subs, &stubGmail{}, NewTokenManager(subs, &stubRefresher{}), NewQueue(1), testOptions())

	if _, err := mail.HandleNotification(context.Background(), "user@example.com", "1", "m1"); err != nil {
		t.Fatalf("notice đầu tiên phải vào queue: %v", err)
	}
	if _, err := mail.HandleNotification(context.Background(), "user@example.com", "2", "m2"); !errors.Is(err, model.ErrQueueFull) {
		t.Fatalf("queue đầy phải trả ErrQueueFull, nhận %v", err)
	}
}

func TestUnsubscribeStopsWatchAndDeletesSubscription(t *testing.T) {
	subs := newStubSubscriptions(model.Subscription{
		ProfileID:            "p1",
		Email:                "user@example.com",
		AccessToken:          "access",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
	})
	mail := NewMailService(subs, &stubGmail{}, NewTokenManager(subs, &stubRefresher{}), NewQueue(4), testOptions())

	if err := mail.Unsubscribe(context.Background(), "p1"); err != nil {
		t.Fatalf("unsubscribe: %v", err)
	}
	if _, err := subs.GetByProfileID(context.Background(), "p1"); !errors.Is(err, model.ErrNotFound) {
		t.Fatalf("subscription phải bị xoá, nhận %v", err)
	}
}
