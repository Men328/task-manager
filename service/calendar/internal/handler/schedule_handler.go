package handler

import (
	"context"

	calendarv1 "taskmanager/common/gen/go/calendar/v1"
	"taskmanager/service/calendar/internal/dependency"
	"taskmanager/service/calendar/internal/service"
)

type ScheduleHandler struct {
	calendarv1.UnimplementedCalendarServiceServer
	schedules service.ScheduleService
}

func NewScheduleHandler(schedules service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{schedules: schedules}
}

func (h *ScheduleHandler) CreateSchedule(ctx context.Context, req *calendarv1.CreateScheduleRequest) (*calendarv1.CreateScheduleResponse, error) {
	if err := dependency.ValidateCreateSchedule(req); err != nil {
		return nil, err
	}

	created, err := h.schedules.Create(ctx, dependency.ScheduleFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &calendarv1.CreateScheduleResponse{Schedule: dependency.ScheduleToProto(created)}, nil
}

func (h *ScheduleHandler) GetSchedule(ctx context.Context, req *calendarv1.GetScheduleRequest) (*calendarv1.GetScheduleResponse, error) {
	if err := dependency.ValidateScheduleID(req.GetId()); err != nil {
		return nil, err
	}

	s, err := h.schedules.Get(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &calendarv1.GetScheduleResponse{Schedule: dependency.ScheduleToProto(s)}, nil
}

func (h *ScheduleHandler) ListSchedules(ctx context.Context, req *calendarv1.ListSchedulesRequest) (*calendarv1.ListSchedulesResponse, error) {
	items, err := h.schedules.List(ctx, dependency.ScheduleFilterFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*calendarv1.Schedule, 0, len(items))
	for _, s := range items {
		out = append(out, dependency.ScheduleToProto(s))
	}
	return &calendarv1.ListSchedulesResponse{Schedules: out}, nil
}

func (h *ScheduleHandler) UpdateSchedule(ctx context.Context, req *calendarv1.UpdateScheduleRequest) (*calendarv1.UpdateScheduleResponse, error) {
	if err := dependency.ValidateScheduleID(req.GetId()); err != nil {
		return nil, err
	}

	updated, err := h.schedules.Update(ctx, req.GetId(), dependency.ScheduleUpdateFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &calendarv1.UpdateScheduleResponse{Schedule: dependency.ScheduleToProto(updated)}, nil
}

func (h *ScheduleHandler) DeleteSchedule(ctx context.Context, req *calendarv1.DeleteScheduleRequest) (*calendarv1.DeleteScheduleResponse, error) {
	if err := dependency.ValidateScheduleID(req.GetId()); err != nil {
		return nil, err
	}

	if err := h.schedules.Delete(ctx, req.GetId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &calendarv1.DeleteScheduleResponse{}, nil
}
