package service

import (
	"context"
	"errors"
	"strings"

	"taskmanager/service/project/internal/model"
)

type projectService struct {
	projects ProjectRepository
}

func NewProjectService(projects ProjectRepository) ProjectService {
	return &projectService{projects: projects}
}

func (s *projectService) Create(ctx context.Context, p model.Project) (model.Project, error) {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return model.Project{}, model.ErrInvalid
	}
	// Đảm bảo project mặc định tồn tại trước để tên "Chung" do user tạo không chiếm chỗ.
	if _, err := s.SeedDefault(ctx, p.ProfileID); err != nil {
		return model.Project{}, err
	}
	p.IsDefault = false
	p.IsArchived = false
	return s.projects.Create(ctx, p)
}

func (s *projectService) Get(ctx context.Context, id string) (model.Project, error) {
	return s.projects.Get(ctx, id)
}

func (s *projectService) List(ctx context.Context, f model.ProjectFilter) ([]model.Project, error) {
	if _, err := s.SeedDefault(ctx, f.ProfileID); err != nil {
		return nil, err
	}
	return s.projects.List(ctx, f)
}

func (s *projectService) Update(ctx context.Context, id string, upd model.ProjectUpdate) (model.Project, error) {
	current, err := s.projects.Get(ctx, id)
	if err != nil {
		return model.Project{}, err
	}
	if current.IsArchived {
		return model.Project{}, model.NewError(model.ErrorKindArchived, "project is archived")
	}
	if upd.Name != nil {
		name := strings.TrimSpace(*upd.Name)
		if name == "" {
			return model.Project{}, model.ErrInvalid
		}
		upd.Name = &name
	}
	return s.projects.Update(ctx, id, upd)
}

func (s *projectService) Archive(ctx context.Context, id string) (model.Project, error) {
	current, err := s.projects.Get(ctx, id)
	if err != nil {
		return model.Project{}, err
	}
	if current.IsDefault {
		return model.Project{}, model.NewError(model.ErrorKindDefaultNotArchivable, "default project cannot be archived")
	}
	return s.projects.SetArchived(ctx, id, true)
}

func (s *projectService) Unarchive(ctx context.Context, id string) (model.Project, error) {
	if _, err := s.projects.Get(ctx, id); err != nil {
		return model.Project{}, err
	}
	return s.projects.SetArchived(ctx, id, false)
}

// SeedDefault trả project mặc định của profile, tạo nếu chưa có (idempotent, an toàn khi race).
func (s *projectService) SeedDefault(ctx context.Context, profileID string) (model.Project, error) {
	d, err := s.projects.GetDefault(ctx, profileID)
	if err == nil {
		return d, nil
	}
	if !errors.Is(err, model.ErrNotFound) {
		return model.Project{}, err
	}
	created, err := s.projects.Create(ctx, model.Project{
		ProfileID: profileID,
		Name:      model.DefaultProjectName,
		IsDefault: true,
	})
	if err != nil {
		// Có thể request khác vừa tạo xong (vi phạm uq_projects_one_default): đọc lại.
		if d, gerr := s.projects.GetDefault(ctx, profileID); gerr == nil {
			return d, nil
		}
		return model.Project{}, err
	}
	return created, nil
}
