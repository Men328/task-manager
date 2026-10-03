package service

import (
	"context"

	"taskmanager/service/calendar/internal/model"
)

type scheduleService struct {
	schedules ScheduleRepository
}

func NewScheduleService(schedules ScheduleRepository) ScheduleService {
	return &scheduleService{schedules: schedules}
}

func (s *scheduleService) Create(ctx context.Context, sc model.Schedule) (model.Schedule, error) {
	if err := validateRange(sc.StartAt, sc.EndAt); err != nil {
		return model.Schedule{}, err
	}
	return s.schedules.Create(ctx, sc)
}

func (s *scheduleService) Get(ctx context.Context, id string) (model.Schedule, error) {
	return s.schedules.Get(ctx, id)
}

func (s *scheduleService) List(ctx context.Context, f model.ScheduleFilter) ([]model.Schedule, error) {
	return s.schedules.List(ctx, f)
}

func (s *scheduleService) Update(ctx context.Context, id string, upd model.ScheduleUpdate) (model.Schedule, error) {
	current, err := s.schedules.Get(ctx, id)
	if err != nil {
		return model.Schedule{}, err
	}

	start := current.StartAt
	if upd.StartAt != nil {
		start = *upd.StartAt
	}
	end := current.EndAt
	if upd.EndAt != nil {
		end = upd.EndAt
	}
	if err := validateRange(start, end); err != nil {
		return model.Schedule{}, err
	}

	return s.schedules.Update(ctx, id, upd)
}

func (s *scheduleService) Delete(ctx context.Context, id string) error {
	return s.schedules.Delete(ctx, id)
}
