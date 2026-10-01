package handler

import (
	"context"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/dependency"
	"taskmanager/service/task/internal/service"
)

type TaskStatusHandler struct {
	taskv1.UnimplementedTaskStatusServiceServer
	statuses service.TaskStatusService
}

func NewTaskStatusHandler(statuses service.TaskStatusService) *TaskStatusHandler {
	return &TaskStatusHandler{statuses: statuses}
}

func (h *TaskStatusHandler) CreateTaskStatus(ctx context.Context, req *taskv1.CreateTaskStatusRequest) (*taskv1.CreateTaskStatusResponse, error) {
	if err := dependency.ValidateCreateTaskStatus(req); err != nil {
		return nil, err
	}

	created, err := h.statuses.Create(ctx, dependency.StatusFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.CreateTaskStatusResponse{Status: dependency.StatusToProto(created)}, nil
}

func (h *TaskStatusHandler) GetTaskStatus(ctx context.Context, req *taskv1.GetTaskStatusRequest) (*taskv1.GetTaskStatusResponse, error) {
	if err := dependency.ValidateTaskStatusID(req.GetId()); err != nil {
		return nil, err
	}

	s, err := h.statuses.Get(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.GetTaskStatusResponse{Status: dependency.StatusToProto(s)}, nil
}

func (h *TaskStatusHandler) ListTaskStatuses(ctx context.Context, req *taskv1.ListTaskStatusesRequest) (*taskv1.ListTaskStatusesResponse, error) {
	items, err := h.statuses.List(ctx, req.GetProfileId(), req.GetIncludeArchived())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*taskv1.TaskStatus, 0, len(items))
	for _, s := range items {
		out = append(out, dependency.StatusToProto(s))
	}
	return &taskv1.ListTaskStatusesResponse{Statuses: out}, nil
}

func (h *TaskStatusHandler) UpdateTaskStatus(ctx context.Context, req *taskv1.UpdateTaskStatusRequest) (*taskv1.UpdateTaskStatusResponse, error) {
	if err := dependency.ValidateTaskStatusID(req.GetId()); err != nil {
		return nil, err
	}

	updated, err := h.statuses.Update(ctx, req.GetId(), dependency.StatusUpdateFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.UpdateTaskStatusResponse{Status: dependency.StatusToProto(updated)}, nil
}

func (h *TaskStatusHandler) DeleteTaskStatus(ctx context.Context, req *taskv1.DeleteTaskStatusRequest) (*taskv1.DeleteTaskStatusResponse, error) {
	if err := dependency.ValidateTaskStatusID(req.GetId()); err != nil {
		return nil, err
	}

	if err := h.statuses.Delete(ctx, req.GetId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &taskv1.DeleteTaskStatusResponse{}, nil
}

func (h *TaskStatusHandler) SeedDefaultStatuses(ctx context.Context, req *taskv1.SeedDefaultStatusesRequest) (*taskv1.SeedDefaultStatusesResponse, error) {
	if err := dependency.ValidateSeedDefaultStatuses(req); err != nil {
		return nil, err
	}

	statuses, transitions, err := h.statuses.SeedDefaultStatuses(ctx, req.GetProfileId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	statusOut := make([]*taskv1.TaskStatus, 0, len(statuses))
	for _, s := range statuses {
		statusOut = append(statusOut, dependency.StatusToProto(s))
	}
	transitionOut := make([]*taskv1.StatusTransition, 0, len(transitions))
	for _, t := range transitions {
		transitionOut = append(transitionOut, dependency.TransitionToProto(t))
	}
	return &taskv1.SeedDefaultStatusesResponse{Statuses: statusOut, Transitions: transitionOut}, nil
}
