package handler

import (
	"context"

	backlogv1 "taskmanager/common/gen/go/backlog/v1"
	"taskmanager/service/backlog/internal/dependency"
	"taskmanager/service/backlog/internal/service"
)

type BacklogHandler struct {
	backlogv1.UnimplementedBacklogServiceServer
	items service.BacklogService
}

func NewBacklogHandler(items service.BacklogService) *BacklogHandler {
	return &BacklogHandler{items: items}
}

func (h *BacklogHandler) CreateBacklog(ctx context.Context, req *backlogv1.CreateBacklogRequest) (*backlogv1.CreateBacklogResponse, error) {
	if err := dependency.ValidateCreateBacklog(req); err != nil {
		return nil, err
	}

	created, err := h.items.Create(ctx, dependency.BacklogFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &backlogv1.CreateBacklogResponse{Backlog: dependency.BacklogToProto(created)}, nil
}

func (h *BacklogHandler) GetBacklog(ctx context.Context, req *backlogv1.GetBacklogRequest) (*backlogv1.GetBacklogResponse, error) {
	if err := dependency.ValidateBacklogID(req.GetId()); err != nil {
		return nil, err
	}

	item, err := h.items.Get(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &backlogv1.GetBacklogResponse{Backlog: dependency.BacklogToProto(item)}, nil
}

func (h *BacklogHandler) ListBacklogs(ctx context.Context, req *backlogv1.ListBacklogsRequest) (*backlogv1.ListBacklogsResponse, error) {
	items, err := h.items.List(ctx, dependency.BacklogFilterFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*backlogv1.Backlog, 0, len(items))
	for _, item := range items {
		out = append(out, dependency.BacklogToProto(item))
	}
	return &backlogv1.ListBacklogsResponse{Backlogs: out}, nil
}

func (h *BacklogHandler) UpdateBacklog(ctx context.Context, req *backlogv1.UpdateBacklogRequest) (*backlogv1.UpdateBacklogResponse, error) {
	if err := dependency.ValidateBacklogID(req.GetId()); err != nil {
		return nil, err
	}

	updated, err := h.items.Update(ctx, req.GetId(), dependency.BacklogUpdateFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &backlogv1.UpdateBacklogResponse{Backlog: dependency.BacklogToProto(updated)}, nil
}

func (h *BacklogHandler) DeleteBacklog(ctx context.Context, req *backlogv1.DeleteBacklogRequest) (*backlogv1.DeleteBacklogResponse, error) {
	if err := dependency.ValidateBacklogID(req.GetId()); err != nil {
		return nil, err
	}

	if err := h.items.Delete(ctx, req.GetId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &backlogv1.DeleteBacklogResponse{}, nil
}
