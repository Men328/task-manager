package dependency

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	workspacev1 "taskmanager/common/gen/go/workspace/v1"
	"taskmanager/service/workspace/internal/model"
)

func ValidateCreateWorkspace(req *workspacev1.CreateWorkspaceRequest) error {
	if strings.TrimSpace(req.GetOwnerProfileId()) == "" {
		return errorcode.Error(errorcode.WorkspaceOwnerProfileIDRequired)
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return errorcode.Error(errorcode.WorkspaceNameRequired)
	}
	if strings.TrimSpace(req.GetSlug()) == "" {
		return errorcode.Error(errorcode.WorkspaceSlugRequired)
	}
	return nil
}

func ValidateWorkspaceID(id string) error {
	if strings.TrimSpace(id) == "" {
		return errorcode.Error(errorcode.WorkspaceIDRequired)
	}
	return nil
}

func WorkspaceFromCreateRequest(req *workspacev1.CreateWorkspaceRequest) model.Workspace {
	return model.Workspace{
		OwnerProfileID: req.GetOwnerProfileId(),
		Name:           req.GetName(),
		Slug:           req.GetSlug(),
		Description:    req.GetDescription(),
		Color:          req.GetColor(),
		Icon:           req.GetIcon(),
		IsDefault:      req.GetIsDefault(),
		Position:       req.GetPosition(),
	}
}

func WorkspaceFilterFromRequest(req *workspacev1.ListWorkspacesRequest) model.WorkspaceFilter {
	return model.WorkspaceFilter{
		OwnerProfileID:  req.GetOwnerProfileId(),
		IncludeArchived: req.GetIncludeArchived(),
	}
}

func WorkspaceUpdateFromRequest(req *workspacev1.UpdateWorkspaceRequest) model.WorkspaceUpdate {
	return model.WorkspaceUpdate{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		Icon:        req.Icon,
		IsDefault:   req.IsDefault,
		Position:    req.Position,
		IsArchived:  req.IsArchived,
	}
}

func WorkspaceToProto(w model.Workspace) *workspacev1.Workspace {
	out := &workspacev1.Workspace{
		Id:             w.ID,
		OwnerProfileId: w.OwnerProfileID,
		Name:           w.Name,
		Slug:           w.Slug,
		Description:    w.Description,
		Color:          w.Color,
		Icon:           w.Icon,
		IsDefault:      w.IsDefault,
		Position:       w.Position,
		IsArchived:     w.IsArchived,
		CreatedAt:      timestamppb.New(w.CreatedAt),
		UpdatedAt:      timestamppb.New(w.UpdatedAt),
	}
	if w.DeletedAt != nil {
		out.DeletedAt = timestamppb.New(*w.DeletedAt)
	}
	return out
}
