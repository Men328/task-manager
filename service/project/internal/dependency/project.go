package dependency

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	projectv1 "taskmanager/common/gen/go/project/v1"
	"taskmanager/service/project/internal/model"
)

func ValidateProfileID(profileID string) error {
	if profileID == "" {
		return errorcode.Error(errorcode.ProjectProfileIDRequired)
	}
	return nil
}

func ValidateCreateProject(req *projectv1.CreateProjectRequest) error {
	if err := ValidateProfileID(req.GetProfileId()); err != nil {
		return err
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return errorcode.Error(errorcode.ProjectNameRequired)
	}
	return nil
}

func ValidateProjectID(id string) error {
	if id == "" {
		return errorcode.Error(errorcode.ProjectIDRequired)
	}
	return nil
}

func ProjectFromCreateRequest(req *projectv1.CreateProjectRequest) model.Project {
	return model.Project{
		ProfileID:   req.GetProfileId(),
		Name:        req.GetName(),
		Description: req.GetDescription(),
		Color:       req.GetColor(),
	}
}

func ProjectFilterFromRequest(req *projectv1.ListProjectsRequest) model.ProjectFilter {
	return model.ProjectFilter{
		ProfileID:       req.GetProfileId(),
		IncludeArchived: req.GetIncludeArchived(),
	}
}

func ProjectUpdateFromRequest(req *projectv1.UpdateProjectRequest) model.ProjectUpdate {
	return model.ProjectUpdate{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
	}
}

func ProjectToProto(p model.Project) *projectv1.Project {
	return &projectv1.Project{
		Id:          p.ID,
		ProfileId:   p.ProfileID,
		Name:        p.Name,
		Description: p.Description,
		Color:       p.Color,
		IsDefault:   p.IsDefault,
		IsArchived:  p.IsArchived,
		CreatedAt:   timestamppb.New(p.CreatedAt),
		UpdatedAt:   timestamppb.New(p.UpdatedAt),
	}
}
