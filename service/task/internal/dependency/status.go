package dependency

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/model"
)

func ValidateCreateTaskStatus(req *taskv1.CreateTaskStatusRequest) error {
	if req.GetProfileId() == "" {
		return errorcode.Error(errorcode.TaskProfileIDRequired, "profile_id là bắt buộc")
	}
	if strings.TrimSpace(req.GetName()) == "" {
		return errorcode.Error(errorcode.TaskStatusNameRequired, "name là bắt buộc")
	}
	if strings.TrimSpace(req.GetSlug()) == "" {
		return errorcode.Error(errorcode.TaskStatusSlugRequired, "slug là bắt buộc")
	}
	return nil
}

func ValidateTaskStatusID(id string) error {
	if id == "" {
		return errorcode.Error(errorcode.TaskStatusIDRequired, "id là bắt buộc")
	}
	return nil
}

func StatusFromCreateRequest(req *taskv1.CreateTaskStatusRequest) model.Status {
	return model.Status{
		ProfileID:   req.GetProfileId(),
		Name:        req.GetName(),
		Slug:        req.GetSlug(),
		Description: req.GetDescription(),
		Color:       req.GetColor(),
		Category:    taskStatusCategoryFromProto(req.GetCategory()),
		IsDefault:   req.GetIsDefault(),
		IsTerminal:  req.GetIsTerminal(),
		Position:    req.GetPosition(),
	}
}

func StatusUpdateFromRequest(req *taskv1.UpdateTaskStatusRequest) model.StatusUpdate {
	return model.StatusUpdate{
		Name:        req.Name,
		Description: req.Description,
		Color:       req.Color,
		Category:    taskStatusCategoryPtr(req.Category),
		IsDefault:   req.IsDefault,
		IsTerminal:  req.IsTerminal,
		Position:    req.Position,
		IsArchived:  req.IsArchived,
	}
}

func StatusToProto(s model.Status) *taskv1.TaskStatus {
	return &taskv1.TaskStatus{
		Id:          s.ID,
		ProfileId:   s.ProfileID,
		Name:        s.Name,
		Slug:        s.Slug,
		Description: s.Description,
		Color:       s.Color,
		Category:    taskStatusCategoryToProto(s.Category),
		IsDefault:   s.IsDefault,
		IsTerminal:  s.IsTerminal,
		Position:    s.Position,
		IsArchived:  s.IsArchived,
		CreatedAt:   timestamppb.New(s.CreatedAt),
		UpdatedAt:   timestamppb.New(s.UpdatedAt),
	}
}

func taskStatusCategoryFromProto(c taskv1.TaskStatusCategory) model.TaskStatusCategory {
	return model.TaskStatusCategory(c)
}

func taskStatusCategoryToProto(c model.TaskStatusCategory) taskv1.TaskStatusCategory {
	return taskv1.TaskStatusCategory(c)
}

func taskStatusCategoryPtr(c *taskv1.TaskStatusCategory) *model.TaskStatusCategory {
	if c == nil {
		return nil
	}
	v := taskStatusCategoryFromProto(*c)
	return &v
}
