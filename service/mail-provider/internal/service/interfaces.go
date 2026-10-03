package service

import (
	"context"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type SubscriptionRepository interface {
	Upsert(ctx context.Context, s model.Subscription) (model.Subscription, error)
	GetByProfileID(ctx context.Context, profileID string) (model.Subscription, error)
	GetByEmail(ctx context.Context, email string) (model.Subscription, error)
	List(ctx context.Context) ([]model.Subscription, error)
	UpdateTokens(ctx context.Context, profileID string, token model.Token) (model.Subscription, error)
	UpdateWatch(ctx context.Context, profileID string, historyID string, watchExpiresAt time.Time) (model.Subscription, error)
	Delete(ctx context.Context, profileID string) error
}

type GmailClient interface {
	Watch(ctx context.Context, accessToken string, topicName string, labelIDs []string) (model.WatchResult, error)
	Stop(ctx context.Context, accessToken string) error
	NewMessageIDs(ctx context.Context, accessToken string, startHistoryID string, labelIDs []string) ([]string, string, error)
	Message(ctx context.Context, accessToken string, messageID string) (model.EmailMessage, error)
	Attachment(ctx context.Context, accessToken string, messageID string, attachmentID string) ([]byte, error)
	ProfileHistoryID(ctx context.Context, accessToken string) (string, error)
}

type TokenRefresher interface {
	Refresh(ctx context.Context, refreshToken string) (model.Token, error)
}

type MailAnalyzer interface {
	Analyze(ctx context.Context, message model.EmailMessage) (model.MailDraft, error)
}

type NotificationSource interface {
	Receive(ctx context.Context) ([]model.PulledNotice, error)
	Acknowledge(ctx context.Context, ackIDs []string) error
}

type TaskCreator interface {
	Create(ctx context.Context, in model.TaskInput) (model.TaskRef, error)
}

type ScheduleCreator interface {
	Create(ctx context.Context, in model.ScheduleInput) (model.ScheduleRef, error)
}

type EventCreator interface {
	Create(ctx context.Context, in model.EventInput) (model.EventRef, error)
}

type BacklogCreator interface {
	Create(ctx context.Context, in model.BacklogInput) (model.BacklogRef, error)
}

type BlobStore interface {
	Put(ctx context.Context, key string, contentType string, data []byte) (string, error)
}

type MailService interface {
	Subscribe(ctx context.Context, in model.SubscribeInput) (model.Subscription, error)
	Unsubscribe(ctx context.Context, profileID string) error
	GetSubscription(ctx context.Context, profileID string) (model.Subscription, error)
	HandleNotification(ctx context.Context, email string, historyID string, messageID string) (bool, error)
}

type Worker interface {
	Run(ctx context.Context) error
}
