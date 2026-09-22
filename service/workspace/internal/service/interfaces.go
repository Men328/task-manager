package service

import (
	"context"

	"taskmanager/service/workspace/internal/model"
)

type WorkspaceRepository interface {
	Create(ctx context.Context, w model.Workspace) (model.Workspace, error)
	Get(ctx context.Context, id string) (model.Workspace, error)
	List(ctx context.Context, f model.WorkspaceFilter) ([]model.Workspace, error)
	Update(ctx context.Context, id string, upd model.WorkspaceUpdate) (model.Workspace, error)
}

type WorkspaceService interface {
	Create(ctx context.Context, w model.Workspace) (model.Workspace, error)
	Get(ctx context.Context, id string) (model.Workspace, error)
	List(ctx context.Context, f model.WorkspaceFilter) ([]model.Workspace, error)
	Update(ctx context.Context, id string, upd model.WorkspaceUpdate) (model.Workspace, error)
}
