package service

import (
	"context"

	"taskmanager/service/backlog/internal/model"
)

type BacklogRepository interface {
	Create(ctx context.Context, item model.Item) (model.Item, error)
	Get(ctx context.Context, id string) (model.Item, error)
	List(ctx context.Context, f model.Filter) ([]model.Item, error)
	Update(ctx context.Context, id string, upd model.Update) (model.Item, error)
	Delete(ctx context.Context, id string) error
}

type BacklogService interface {
	Create(ctx context.Context, item model.Item) (model.Item, error)
	Get(ctx context.Context, id string) (model.Item, error)
	List(ctx context.Context, f model.Filter) ([]model.Item, error)
	Update(ctx context.Context, id string, upd model.Update) (model.Item, error)
	Delete(ctx context.Context, id string) error
}

func normalizeStatus(status model.Status, fallback model.Status) (model.Status, error) {
	if status == "" {
		return fallback, nil
	}
	if !status.Valid() {
		return "", model.NewError(model.ErrorKindStatusInvalid, "unsupported backlog status %q", status)
	}
	return status, nil
}

func normalizeCategory(category string) string {
	if category == "" {
		return model.CategoryOther
	}
	return category
}
