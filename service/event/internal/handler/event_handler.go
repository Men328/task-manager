package handler

import (
	"context"

	eventv1 "taskmanager/common/gen/go/event/v1"
	"taskmanager/service/event/internal/dependency"
	"taskmanager/service/event/internal/service"
)

type EventHandler struct {
	eventv1.UnimplementedEventServiceServer
	events service.EventService
}

func NewEventHandler(events service.EventService) *EventHandler {
	return &EventHandler{events: events}
}

func (h *EventHandler) CreateEvent(ctx context.Context, req *eventv1.CreateEventRequest) (*eventv1.CreateEventResponse, error) {
	if err := dependency.ValidateCreateEvent(req); err != nil {
		return nil, err
	}

	created, err := h.events.Create(ctx, dependency.EventFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &eventv1.CreateEventResponse{Event: dependency.EventToProto(created)}, nil
}

func (h *EventHandler) GetEvent(ctx context.Context, req *eventv1.GetEventRequest) (*eventv1.GetEventResponse, error) {
	if err := dependency.ValidateEventID(req.GetId()); err != nil {
		return nil, err
	}

	e, err := h.events.Get(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &eventv1.GetEventResponse{Event: dependency.EventToProto(e)}, nil
}

func (h *EventHandler) ListEvents(ctx context.Context, req *eventv1.ListEventsRequest) (*eventv1.ListEventsResponse, error) {
	items, err := h.events.List(ctx, dependency.EventFilterFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*eventv1.Event, 0, len(items))
	for _, e := range items {
		out = append(out, dependency.EventToProto(e))
	}
	return &eventv1.ListEventsResponse{Events: out}, nil
}

func (h *EventHandler) UpdateEvent(ctx context.Context, req *eventv1.UpdateEventRequest) (*eventv1.UpdateEventResponse, error) {
	if err := dependency.ValidateEventID(req.GetId()); err != nil {
		return nil, err
	}

	updated, err := h.events.Update(ctx, req.GetId(), dependency.EventUpdateFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &eventv1.UpdateEventResponse{Event: dependency.EventToProto(updated)}, nil
}

func (h *EventHandler) DeleteEvent(ctx context.Context, req *eventv1.DeleteEventRequest) (*eventv1.DeleteEventResponse, error) {
	if err := dependency.ValidateEventID(req.GetId()); err != nil {
		return nil, err
	}

	if err := h.events.Delete(ctx, req.GetId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &eventv1.DeleteEventResponse{}, nil
}
