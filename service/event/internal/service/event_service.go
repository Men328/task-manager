package service

import (
	"context"
	"strings"

	"taskmanager/service/event/internal/model"
)

type eventService struct {
	events   EventRepository
	projects ProjectChecker
}

func NewEventService(events EventRepository, projects ProjectChecker) EventService {
	return &eventService{events: events, projects: projects}
}

func (s *eventService) Create(ctx context.Context, e model.Event) (model.Event, error) {
	if err := validateRange(e.StartAt, e.EndAt); err != nil {
		return model.Event{}, err
	}
	status, err := normalizeStatus(e.Status, model.StatusPlanned)
	if err != nil {
		return model.Event{}, err
	}
	e.Status = status
	if e.ProjectID == "" {
		return model.Event{}, model.NewError(model.ErrorKindProjectRequired, "project_id is required")
	}
	if err := s.projects.CheckAssignable(ctx, e.ProfileID, e.ProjectID); err != nil {
		return model.Event{}, err
	}
	return s.events.Create(ctx, e)
}

func (s *eventService) Get(ctx context.Context, id string) (model.Event, error) {
	return s.events.Get(ctx, id)
}

func (s *eventService) List(ctx context.Context, f model.Filter) ([]model.Event, error) {
	return s.events.List(ctx, f)
}

func (s *eventService) Update(ctx context.Context, id string, upd model.Update) (model.Event, error) {
	current, err := s.events.Get(ctx, id)
	if err != nil {
		return model.Event{}, err
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
		return model.Event{}, err
	}
	if upd.Status != nil {
		status, err := normalizeStatus(*upd.Status, current.Status)
		if err != nil {
			return model.Event{}, err
		}
		upd.Status = &status
	}

	if upd.ProjectID != nil {
		if strings.TrimSpace(*upd.ProjectID) == "" {
			return model.Event{}, model.NewError(model.ErrorKindProjectRequired, "project_id must not be empty")
		}
		if err := s.projects.CheckAssignable(ctx, current.ProfileID, *upd.ProjectID); err != nil {
			return model.Event{}, err
		}
	}

	return s.events.Update(ctx, id, upd)
}

func (s *eventService) Delete(ctx context.Context, id string) error {
	return s.events.Delete(ctx, id)
}
