package service

import (
	"context"
	"strings"

	"taskmanager/service/calendar/internal/model"
)

type scheduleService struct {
	schedules ScheduleRepository
	projects  ProjectChecker
}

func NewScheduleService(schedules ScheduleRepository, projects ProjectChecker) ScheduleService {
	return &scheduleService{schedules: schedules, projects: projects}
}

func (s *scheduleService) Create(ctx context.Context, sc model.Schedule) (model.Schedule, error) {
	if err := validateRange(sc.StartAt, sc.EndAt); err != nil {
		return model.Schedule{}, err
	}
	if strings.TrimSpace(sc.ProjectID) == "" {
		return model.Schedule{}, model.NewError(model.ErrorKindProjectRequired, "project_id is required")
	}
	if err := s.projects.CheckAssignable(ctx, sc.ProfileID, sc.ProjectID); err != nil {
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

	if upd.ProjectID != nil {
		if strings.TrimSpace(*upd.ProjectID) == "" {
			return model.Schedule{}, model.NewError(model.ErrorKindProjectRequired, "project_id must not be empty")
		}
		if err := s.projects.CheckAssignable(ctx, current.ProfileID, *upd.ProjectID); err != nil {
			return model.Schedule{}, err
		}
	}

	return s.schedules.Update(ctx, id, upd)
}

func (s *scheduleService) Delete(ctx context.Context, id string) error {
	return s.schedules.Delete(ctx, id)
}
