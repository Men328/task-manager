package service

import (
	"context"
	"time"

	"taskmanager/service/event/internal/model"
)

type EventRepository interface {
	Create(ctx context.Context, e model.Event) (model.Event, error)
	Get(ctx context.Context, id string) (model.Event, error)
	List(ctx context.Context, f model.Filter) ([]model.Event, error)
	Update(ctx context.Context, id string, upd model.Update) (model.Event, error)
	Delete(ctx context.Context, id string) error
}

type EventService interface {
	Create(ctx context.Context, e model.Event) (model.Event, error)
	Get(ctx context.Context, id string) (model.Event, error)
	List(ctx context.Context, f model.Filter) ([]model.Event, error)
	Update(ctx context.Context, id string, upd model.Update) (model.Event, error)
	Delete(ctx context.Context, id string) error
}

func validateRange(start time.Time, end *time.Time) error {
	if end != nil && end.Before(start) {
		return model.NewError(model.ErrorKindTimeRangeInvalid, "end time must not be before start time")
	}
	return nil
}

func normalizeStatus(status model.Status, fallback model.Status) (model.Status, error) {
	if status == "" {
		return fallback, nil
	}
	if !status.Valid() {
		return "", model.NewError(model.ErrorKindStatusInvalid, "unsupported event status %q", status)
	}
	return status, nil
}
