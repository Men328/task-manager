package service

import (
	"context"

	"taskmanager/service/workspace/internal/model"
)

type workspaceService struct {
	workspaces WorkspaceRepository
}

func NewWorkspaceService(workspaces WorkspaceRepository) WorkspaceService {
	return &workspaceService{workspaces: workspaces}
}

func (s *workspaceService) Create(ctx context.Context, w model.Workspace) (model.Workspace, error) {
	return s.workspaces.Create(ctx, w)
}

func (s *workspaceService) Get(ctx context.Context, id string) (model.Workspace, error) {
	return s.workspaces.Get(ctx, id)
}

func (s *workspaceService) List(ctx context.Context, f model.WorkspaceFilter) ([]model.Workspace, error) {
	return s.workspaces.List(ctx, f)
}

func (s *workspaceService) Update(ctx context.Context, id string, upd model.WorkspaceUpdate) (model.Workspace, error) {
	return s.workspaces.Update(ctx, id, upd)
}
