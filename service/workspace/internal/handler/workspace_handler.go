package handler

import (
	"context"

	workspacev1 "taskmanager/common/gen/go/workspace/v1"
	"taskmanager/service/workspace/internal/dependency"
	"taskmanager/service/workspace/internal/service"
)

type WorkspaceHandler struct {
	workspacev1.UnimplementedWorkspaceServiceServer
	workspaces service.WorkspaceService
}

func NewWorkspaceHandler(workspaces service.WorkspaceService) *WorkspaceHandler {
	return &WorkspaceHandler{workspaces: workspaces}
}

func (h *WorkspaceHandler) CreateWorkspace(ctx context.Context, req *workspacev1.CreateWorkspaceRequest) (*workspacev1.CreateWorkspaceResponse, error) {
	if err := dependency.ValidateCreateWorkspace(req); err != nil {
		return nil, err
	}

	created, err := h.workspaces.Create(ctx, dependency.WorkspaceFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &workspacev1.CreateWorkspaceResponse{Workspace: dependency.WorkspaceToProto(created)}, nil
}

func (h *WorkspaceHandler) GetWorkspace(ctx context.Context, req *workspacev1.GetWorkspaceRequest) (*workspacev1.GetWorkspaceResponse, error) {
	if err := dependency.ValidateWorkspaceID(req.GetId()); err != nil {
		return nil, err
	}

	w, err := h.workspaces.Get(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &workspacev1.GetWorkspaceResponse{Workspace: dependency.WorkspaceToProto(w)}, nil
}

func (h *WorkspaceHandler) ListWorkspaces(ctx context.Context, req *workspacev1.ListWorkspacesRequest) (*workspacev1.ListWorkspacesResponse, error) {
	items, err := h.workspaces.List(ctx, dependency.WorkspaceFilterFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}

	out := make([]*workspacev1.Workspace, 0, len(items))
	for _, w := range items {
		out = append(out, dependency.WorkspaceToProto(w))
	}
	return &workspacev1.ListWorkspacesResponse{Workspaces: out}, nil
}

func (h *WorkspaceHandler) UpdateWorkspace(ctx context.Context, req *workspacev1.UpdateWorkspaceRequest) (*workspacev1.UpdateWorkspaceResponse, error) {
	if err := dependency.ValidateWorkspaceID(req.GetId()); err != nil {
		return nil, err
	}

	updated, err := h.workspaces.Update(ctx, req.GetId(), dependency.WorkspaceUpdateFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &workspacev1.UpdateWorkspaceResponse{Workspace: dependency.WorkspaceToProto(updated)}, nil
}
