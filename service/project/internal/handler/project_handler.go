package handler

import (
	"context"

	projectv1 "taskmanager/common/gen/go/project/v1"
	"taskmanager/service/project/internal/dependency"
	"taskmanager/service/project/internal/service"
)

type ProjectHandler struct {
	projectv1.UnimplementedProjectServiceServer
	projects service.ProjectService
}

func NewProjectHandler(projects service.ProjectService) *ProjectHandler {
	return &ProjectHandler{projects: projects}
}

func (h *ProjectHandler) CreateProject(ctx context.Context, req *projectv1.CreateProjectRequest) (*projectv1.CreateProjectResponse, error) {
	if err := dependency.ValidateCreateProject(req); err != nil {
		return nil, err
	}
	p, err := h.projects.Create(ctx, dependency.ProjectFromCreateRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &projectv1.CreateProjectResponse{Project: dependency.ProjectToProto(p)}, nil
}

func (h *ProjectHandler) GetProject(ctx context.Context, req *projectv1.GetProjectRequest) (*projectv1.GetProjectResponse, error) {
	if err := dependency.ValidateProjectID(req.GetId()); err != nil {
		return nil, err
	}
	p, err := h.projects.Get(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &projectv1.GetProjectResponse{Project: dependency.ProjectToProto(p)}, nil
}

func (h *ProjectHandler) ListProjects(ctx context.Context, req *projectv1.ListProjectsRequest) (*projectv1.ListProjectsResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}
	items, err := h.projects.List(ctx, dependency.ProjectFilterFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	out := make([]*projectv1.Project, 0, len(items))
	for _, p := range items {
		out = append(out, dependency.ProjectToProto(p))
	}
	return &projectv1.ListProjectsResponse{Projects: out}, nil
}

func (h *ProjectHandler) UpdateProject(ctx context.Context, req *projectv1.UpdateProjectRequest) (*projectv1.UpdateProjectResponse, error) {
	if err := dependency.ValidateProjectID(req.GetId()); err != nil {
		return nil, err
	}
	p, err := h.projects.Update(ctx, req.GetId(), dependency.ProjectUpdateFromRequest(req))
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &projectv1.UpdateProjectResponse{Project: dependency.ProjectToProto(p)}, nil
}

func (h *ProjectHandler) ArchiveProject(ctx context.Context, req *projectv1.ArchiveProjectRequest) (*projectv1.ArchiveProjectResponse, error) {
	if err := dependency.ValidateProjectID(req.GetId()); err != nil {
		return nil, err
	}
	p, err := h.projects.Archive(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &projectv1.ArchiveProjectResponse{Project: dependency.ProjectToProto(p)}, nil
}

func (h *ProjectHandler) UnarchiveProject(ctx context.Context, req *projectv1.UnarchiveProjectRequest) (*projectv1.UnarchiveProjectResponse, error) {
	if err := dependency.ValidateProjectID(req.GetId()); err != nil {
		return nil, err
	}
	p, err := h.projects.Unarchive(ctx, req.GetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &projectv1.UnarchiveProjectResponse{Project: dependency.ProjectToProto(p)}, nil
}

func (h *ProjectHandler) SeedDefaultProject(ctx context.Context, req *projectv1.SeedDefaultProjectRequest) (*projectv1.SeedDefaultProjectResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}
	p, err := h.projects.SeedDefault(ctx, req.GetProfileId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &projectv1.SeedDefaultProjectResponse{Project: dependency.ProjectToProto(p)}, nil
}
