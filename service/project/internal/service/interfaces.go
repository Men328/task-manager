package service

import (
	"context"

	"taskmanager/service/project/internal/model"
)

type ProjectRepository interface {
	Create(ctx context.Context, p model.Project) (model.Project, error)
	Get(ctx context.Context, id string) (model.Project, error)
	List(ctx context.Context, f model.ProjectFilter) ([]model.Project, error)
	// GetDefault trả model.ErrNotFound nếu profile chưa có project mặc định.
	GetDefault(ctx context.Context, profileID string) (model.Project, error)
	Update(ctx context.Context, id string, upd model.ProjectUpdate) (model.Project, error)
	SetArchived(ctx context.Context, id string, archived bool) (model.Project, error)
}

type ProjectService interface {
	Create(ctx context.Context, p model.Project) (model.Project, error)
	Get(ctx context.Context, id string) (model.Project, error)
	List(ctx context.Context, f model.ProjectFilter) ([]model.Project, error)
	Update(ctx context.Context, id string, upd model.ProjectUpdate) (model.Project, error)
	Archive(ctx context.Context, id string) (model.Project, error)
	Unarchive(ctx context.Context, id string) (model.Project, error)
	SeedDefault(ctx context.Context, profileID string) (model.Project, error)
}
