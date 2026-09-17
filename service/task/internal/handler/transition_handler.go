package handler

import (
	"context"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/dependency"
	"taskmanager/service/task/internal/service"
)

type StatusTransitionHandler struct {
	taskv1.UnimplementedStatusTransitionServiceServer
	transitions service.StatusTransitionService
}

func NewStatusTransitionHandler(transitions service.StatusTransitionService) *StatusTransitionHandler {
	return &StatusTransitionHandler{transitions: transitions}
}

func (h *StatusTransitionHandler) CreateStatusTransition(ctx context.Context, req *taskv1.CreateStatusTransitionRequest) (*taskv1.CreateStatusTransitionResponse, error) {
	if err := dependency.ValidateCreateStatusTransition(req); err != nil {
		return nil, err
	}

	created, err := h.transitions.Create(ctx, dependency.TransitionFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.CreateStatusTransitionResponse{Transition: dependency.TransitionToProto(created)}, nil
}

func (h *StatusTransitionHandler) GetStatusTransition(ctx context.Context, req *taskv1.GetStatusTransitionRequest) (*taskv1.GetStatusTransitionResponse, error) {
	if err := dependency.ValidateStatusTransitionID(req.GetId()); err != nil {
		return nil, err
	}

	t, err := h.transitions.Get(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.GetStatusTransitionResponse{Transition: dependency.TransitionToProto(t)}, nil
}

func (h *StatusTransitionHandler) ListStatusTransitions(ctx context.Context, req *taskv1.ListStatusTransitionsRequest) (*taskv1.ListStatusTransitionsResponse, error) {
	items, err := h.transitions.List(ctx, req.GetProfileId(), req.GetFromStatusId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*taskv1.StatusTransition, 0, len(items))
	for _, t := range items {
		out = append(out, dependency.TransitionToProto(t))
	}
	return &taskv1.ListStatusTransitionsResponse{Transitions: out}, nil
}

func (h *StatusTransitionHandler) UpdateStatusTransition(ctx context.Context, req *taskv1.UpdateStatusTransitionRequest) (*taskv1.UpdateStatusTransitionResponse, error) {
	if err := dependency.ValidateStatusTransitionID(req.GetId()); err != nil {
		return nil, err
	}

	updated, err := h.transitions.Update(ctx, req.GetId(), dependency.TransitionUpdateFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.UpdateStatusTransitionResponse{Transition: dependency.TransitionToProto(updated)}, nil
}

func (h *StatusTransitionHandler) DeleteStatusTransition(ctx context.Context, req *taskv1.DeleteStatusTransitionRequest) (*taskv1.DeleteStatusTransitionResponse, error) {
	if err := dependency.ValidateStatusTransitionID(req.GetId()); err != nil {
		return nil, err
	}

	if err := h.transitions.Delete(ctx, req.GetId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.DeleteStatusTransitionResponse{}, nil
}

func (h *StatusTransitionHandler) ValidateStatusTransition(ctx context.Context, req *taskv1.ValidateStatusTransitionRequest) (*taskv1.ValidateStatusTransitionResponse, error) {
	if err := dependency.ValidateStatusTransitionRequest(req); err != nil {
		return nil, err
	}

	allowed, reason, err := h.transitions.ValidateStatusTransition(
		ctx, req.GetProfileId(), req.GetFromStatusId(), req.GetToStatusId(),
	)
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.ValidateStatusTransitionResponse{Allowed: allowed, Reason: reason}, nil
}
