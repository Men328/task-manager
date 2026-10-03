package dependency

import (
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	calendarv1 "taskmanager/common/gen/go/calendar/v1"
	"taskmanager/service/calendar/internal/model"
)

func ValidateCreateSchedule(req *calendarv1.CreateScheduleRequest) error {
	if req.GetProfileId() == "" {
		return errorcode.Error(errorcode.CalendarProfileIDRequired)
	}
	if strings.TrimSpace(req.GetTitle()) == "" {
		return errorcode.Error(errorcode.CalendarTitleRequired)
	}
	if req.GetStartAt() == nil || !req.GetStartAt().IsValid() {
		return errorcode.Error(errorcode.CalendarStartAtRequired)
	}
	return nil
}

func ValidateScheduleID(id string) error {
	if id == "" {
		return errorcode.Error(errorcode.CalendarIDRequired)
	}
	return nil
}

func ScheduleFromCreateRequest(req *calendarv1.CreateScheduleRequest) model.Schedule {
	return model.Schedule{
		ProfileID:   req.GetProfileId(),
		Title:       req.GetTitle(),
		Description: req.GetDescription(),
		Location:    req.GetLocation(),
		StartAt:     timeValue(req.GetStartAt()),
		EndAt:       timePtr(req.GetEndAt()),
		AllDay:      req.GetAllDay(),
		Color:       req.GetColor(),
	}
}

func ScheduleFilterFromRequest(req *calendarv1.ListSchedulesRequest) model.ScheduleFilter {
	return model.ScheduleFilter{
		ProfileID: req.GetProfileId(),
		From:      timePtr(req.GetFrom()),
		To:        timePtr(req.GetTo()),
	}
}

func ScheduleUpdateFromRequest(req *calendarv1.UpdateScheduleRequest) model.ScheduleUpdate {
	return model.ScheduleUpdate{
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		StartAt:     timePtr(req.GetStartAt()),
		EndAt:       timePtr(req.GetEndAt()),
		AllDay:      req.AllDay,
		Color:       req.Color,
	}
}

func ScheduleToProto(s model.Schedule) *calendarv1.Schedule {
	return &calendarv1.Schedule{
		Id:          s.ID,
		ProfileId:   s.ProfileID,
		Title:       s.Title,
		Description: s.Description,
		Location:    s.Location,
		StartAt:     timestamppb.New(s.StartAt),
		EndAt:       timestampPtr(s.EndAt),
		AllDay:      s.AllDay,
		Color:       s.Color,
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
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
