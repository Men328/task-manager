package dependency

import (
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/model"
)

func ValidateCreateTask(req *taskv1.CreateTaskRequest) error {
	if req.GetProfileId() == "" {
		return status.Error(codes.InvalidArgument, "profile_id là bắt buộc")
	}
	if strings.TrimSpace(req.GetTitle()) == "" {
		return status.Error(codes.InvalidArgument, "title là bắt buộc")
	}
	return nil
}

func ValidateTaskID(id string) error {
	if id == "" {
		return status.Error(codes.InvalidArgument, "id là bắt buộc")
	}
	return nil
}

func ValidateChangeTaskStatus(req *taskv1.ChangeTaskStatusRequest) error {
	if req.GetId() == "" || req.GetStatusId() == "" {
		return status.Error(codes.InvalidArgument, "id và status_id là bắt buộc")
	}
	return nil
}

func TaskFromCreateRequest(req *taskv1.CreateTaskRequest) model.Task {
	return model.Task{
		ProfileID:    req.GetProfileId(),
		ParentTaskID: req.GetParentTaskId(),
		StatusID:     req.GetStatusId(),
		Title:        req.GetTitle(),
		Description:  req.GetDescription(),
		Priority:     taskPriorityFromProto(req.GetPriority()),
		Position:     req.GetPosition(),
		StartAt:      timePtr(req.GetStartAt()),
		DueAt:        timePtr(req.GetDueAt()),
	}
}

func TaskFilterFromRequest(req *taskv1.ListTasksRequest) model.TaskFilter {
	return model.TaskFilter{
		ProfileID:       req.GetProfileId(),
		ParentTaskID:    req.GetParentTaskId(),
		RootOnly:        req.GetRootOnly(),
		StatusID:        req.GetStatusId(),
		IncludeArchived: req.GetIncludeArchived(),
	}
}

func TaskUpdateFromRequest(req *taskv1.UpdateTaskRequest) model.TaskUpdate {
	return model.TaskUpdate{
		ParentTaskID: req.ParentTaskId,
		Title:        req.Title,
		Description:  req.Description,
		Priority:     taskPriorityPtr(req.Priority),
		Position:     req.Position,
		StartAt:      timePtr(req.GetStartAt()),
		DueAt:        timePtr(req.GetDueAt()),
		IsArchived:   req.IsArchived,
	}
}

func TaskToProto(t model.Task) *taskv1.Task {
	return &taskv1.Task{
		Id:           t.ID,
		ProfileId:    t.ProfileID,
		ParentTaskId: t.ParentTaskID,
		StatusId:     t.StatusID,
		Title:        t.Title,
		Description:  t.Description,
		Priority:     taskPriorityToProto(t.Priority),
		Position:     t.Position,
		StartAt:      timestamp(t.StartAt),
		DueAt:        timestamp(t.DueAt),
		CompletedAt:  timestamp(t.CompletedAt),
		IsArchived:   t.IsArchived,
		CreatedAt:    timestamppb.New(t.CreatedAt),
		UpdatedAt:    timestamppb.New(t.UpdatedAt),
	}
}

func taskPriorityFromProto(p taskv1.TaskPriority) model.TaskPriority {
	return model.TaskPriority(p)
}

func taskPriorityToProto(p model.TaskPriority) taskv1.TaskPriority {
	return taskv1.TaskPriority(p)
}

func taskPriorityPtr(p *taskv1.TaskPriority) *model.TaskPriority {
	if p == nil {
		return nil
	}
	v := taskPriorityFromProto(*p)
	return &v
}

func timestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func timePtr(ts *timestamppb.Timestamp) *time.Time {
	if ts == nil {
		return nil
	}
	t := ts.AsTime().UTC()
	return &t
}
