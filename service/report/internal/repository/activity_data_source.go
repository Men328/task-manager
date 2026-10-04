package repository

import (
	"context"
	"fmt"
	"time"

	backlogv1 "taskmanager/common/gen/go/backlog/v1"
	calendarv1 "taskmanager/common/gen/go/calendar/v1"
	eventv1 "taskmanager/common/gen/go/event/v1"
	"taskmanager/service/report/internal/model"
)

type ActivityDataSource struct {
	events    eventv1.EventServiceClient
	schedules calendarv1.CalendarServiceClient
	backlogs  backlogv1.BacklogServiceClient
	timeout   time.Duration
}

func NewActivityDataSource(
	events eventv1.EventServiceClient,
	schedules calendarv1.CalendarServiceClient,
	backlogs backlogv1.BacklogServiceClient,
	timeout time.Duration,
) *ActivityDataSource {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &ActivityDataSource{events: events, schedules: schedules, backlogs: backlogs, timeout: timeout}
}

func (s *ActivityDataSource) FetchEvents(ctx context.Context, profileID string) ([]model.Event, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	response, err := s.events.ListEvents(callCtx, &eventv1.ListEventsRequest{ProfileId: profileID})
	if err != nil {
		return nil, fmt.Errorf("list events cho profile %s: %w", profileID, err)
	}

	out := make([]model.Event, 0, len(response.GetEvents()))
	for _, event := range response.GetEvents() {
		out = append(out, model.Event{
			ID:      event.GetId(),
			Status:  eventStatusToModel(event.GetStatus()),
			StartAt: timeValue(event.GetStartAt()),
			EndAt:   timePtr(event.GetEndAt()),
			AllDay:  event.GetAllDay(),
		})
	}
	return out, nil
}

func (s *ActivityDataSource) FetchSchedules(ctx context.Context, profileID string) ([]model.Schedule, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	response, err := s.schedules.ListSchedules(callCtx, &calendarv1.ListSchedulesRequest{ProfileId: profileID})
	if err != nil {
		return nil, fmt.Errorf("list schedules cho profile %s: %w", profileID, err)
	}

	out := make([]model.Schedule, 0, len(response.GetSchedules()))
	for _, schedule := range response.GetSchedules() {
		out = append(out, model.Schedule{
			ID:      schedule.GetId(),
			StartAt: timeValue(schedule.GetStartAt()),
			EndAt:   timePtr(schedule.GetEndAt()),
			AllDay:  schedule.GetAllDay(),
		})
	}
	return out, nil
}

func (s *ActivityDataSource) FetchBacklogs(ctx context.Context, profileID string) ([]model.BacklogItem, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	response, err := s.backlogs.ListBacklogs(callCtx, &backlogv1.ListBacklogsRequest{ProfileId: profileID})
	if err != nil {
		return nil, fmt.Errorf("list backlogs cho profile %s: %w", profileID, err)
	}

	out := make([]model.BacklogItem, 0, len(response.GetBacklogs()))
	for _, item := range response.GetBacklogs() {
		out = append(out, model.BacklogItem{
			ID:        item.GetId(),
			Status:    backlogStatusToModel(item.GetStatus()),
			Category:  item.GetCategory(),
			CreatedAt: timeValue(item.GetCreatedAt()),
		})
	}
	return out, nil
}

func eventStatusToModel(status eventv1.EventStatus) string {
	switch status {
	case eventv1.EventStatus_EVENT_STATUS_PLANNED:
		return model.EventStatusPlanned
	case eventv1.EventStatus_EVENT_STATUS_CONFIRMED:
		return model.EventStatusConfirmed
	case eventv1.EventStatus_EVENT_STATUS_CANCELLED:
		return model.EventStatusCancelled
	default:
		return ""
	}
}

func backlogStatusToModel(status backlogv1.BacklogStatus) string {
	switch status {
	case backlogv1.BacklogStatus_BACKLOG_STATUS_NEW:
		return model.BacklogStatusNew
	case backlogv1.BacklogStatus_BACKLOG_STATUS_TRIAGED:
		return model.BacklogStatusTriaged
	case backlogv1.BacklogStatus_BACKLOG_STATUS_ARCHIVED:
		return model.BacklogStatusArchived
	default:
		return ""
	}
}
