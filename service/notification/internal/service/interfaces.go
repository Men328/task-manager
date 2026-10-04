package service

import (
	"context"

	"taskmanager/service/notification/internal/model"
)

type NoticeRepository interface {
	Create(ctx context.Context, notice model.Notice) (model.Notice, error)
	Get(ctx context.Context, id string) (model.Notice, error)
	List(ctx context.Context, filter model.Filter) ([]model.Notice, error)
	CountUnread(ctx context.Context, profileID string) (int, error)
	MarkRead(ctx context.Context, profileID string, ids []string) (int, error)
}

type NoticePublisher interface {
	Publish(ctx context.Context, notice model.Notice) error
}

type NoticeService interface {
	Create(ctx context.Context, notice model.Notice) (model.Notice, error)
	List(ctx context.Context, filter model.Filter) ([]model.Notice, error)
	CountUnread(ctx context.Context, profileID string) (int, error)
	MarkRead(ctx context.Context, profileID string, ids []string) (int, error)
}
