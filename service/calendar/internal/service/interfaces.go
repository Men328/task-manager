package service

import (
	"context"
	"time"

	"taskmanager/service/calendar/internal/model"
)

type ScheduleRepository interface {
	Create(ctx context.Context, s model.Schedule) (model.Schedule, error)
	Get(ctx context.Context, id string) (model.Schedule, error)
	List(ctx context.Context, f model.ScheduleFilter) ([]model.Schedule, error)
	Update(ctx context.Context, id string, upd model.ScheduleUpdate) (model.Schedule, error)
	Delete(ctx context.Context, id string) error
}

type ScheduleService interface {
	Create(ctx context.Context, s model.Schedule) (model.Schedule, error)
	Get(ctx context.Context, id string) (model.Schedule, error)
	List(ctx context.Context, f model.ScheduleFilter) ([]model.Schedule, error)
	Update(ctx context.Context, id string, upd model.ScheduleUpdate) (model.Schedule, error)
	Delete(ctx context.Context, id string) error
}

func validateRange(start time.Time, end *time.Time) error {
	if end != nil && end.Before(start) {
		return model.NewError(model.ErrorKindTimeRangeInvalid, "end time must not be before start time")
	}
	return nil
}
