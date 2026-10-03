package dependency

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	eventv1 "taskmanager/common/gen/go/event/v1"
	"taskmanager/service/event/internal/model"
)

func ValidateCreateEvent(req *eventv1.CreateEventRequest) error {
	if req.GetProfileId() == "" {
		return errorcode.Error(errorcode.EventProfileIDRequired)
	}
	if strings.TrimSpace(req.GetTitle()) == "" {
		return errorcode.Error(errorcode.EventTitleRequired)
	}
	if req.GetStartAt() == nil || !req.GetStartAt().IsValid() {
		return errorcode.Error(errorcode.EventStartAtRequired)
	}
	return nil
}

func ValidateEventID(id string) error {
	if id == "" {
		return errorcode.Error(errorcode.EventIDRequired)
	}
	return nil
}

func EventFromCreateRequest(req *eventv1.CreateEventRequest) model.Event {
	return model.Event{
		ProfileID:   req.GetProfileId(),
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Location:    req.GetLocation(),
		StartAt:     timeValue(req.GetStartAt()),
		EndAt:       timePtr(req.GetEndAt()),
		AllDay:      req.GetAllDay(),
		Color:       req.GetColor(),
		Status:      statusFromProto(req.GetStatus()),
		Source:      req.GetSource(),
	}
}

func EventFilterFromRequest(req *eventv1.ListEventsRequest) model.Filter {
	return model.Filter{
		ProfileID: req.GetProfileId(),
		From:      timePtr(req.GetFrom()),
		To:        timePtr(req.GetTo()),
		Status:    statusFromProto(req.GetStatus()),
	}
}

func EventUpdateFromRequest(req *eventv1.UpdateEventRequest) model.Update {
	return model.Update{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		StartAt:     timePtr(req.GetStartAt()),
		EndAt:       timePtr(req.GetEndAt()),
		AllDay:      req.AllDay,
		Color:       req.Color,
		Status:      updateStatusFromProto(req.Status),
	}
}

func EventToProto(e model.Event) *eventv1.Event {
	return &eventv1.Event{
		Id:          e.ID,
		ProfileId:   e.ProfileID,
		Title:       e.Title,
		Description: e.Description,
		Location:    e.Location,
		StartAt:     timestamppb.New(e.StartAt),
		EndAt:       timestampPtr(e.EndAt),
		AllDay:      e.AllDay,
		Color:       e.Color,
		Status:      statusToProto(e.Status),
		Source:      e.Source,
		CreatedAt:   timestamppb.New(e.CreatedAt),
		UpdatedAt:   timestamppb.New(e.UpdatedAt),
	}
}

func statusToProto(status model.Status) eventv1.EventStatus {
	switch status {
	case model.StatusPlanned:
		return eventv1.EventStatus_EVENT_STATUS_PLANNED
	case model.StatusConfirmed:
		return eventv1.EventStatus_EVENT_STATUS_CONFIRMED
	case model.StatusCancelled:
		return eventv1.EventStatus_EVENT_STATUS_CANCELLED
	default:
		return eventv1.EventStatus_EVENT_STATUS_UNSPECIFIED
	}
}

func statusFromProto(status eventv1.EventStatus) model.Status {
	switch status {
	case eventv1.EventStatus_EVENT_STATUS_PLANNED:
		return model.StatusPlanned
	case eventv1.EventStatus_EVENT_STATUS_CONFIRMED:
		return model.StatusConfirmed
	case eventv1.EventStatus_EVENT_STATUS_CANCELLED:
		return model.StatusCancelled
	default:
		return ""
	}
}

func updateStatusFromProto(status *eventv1.EventStatus) *model.Status {
	if status == nil {
		return nil
	}
	converted := statusFromProto(*status)
	return &converted
}

func timePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime().UTC()
	return &t
}

func timeValue(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime().UTC()
}

func timestampPtr(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}
