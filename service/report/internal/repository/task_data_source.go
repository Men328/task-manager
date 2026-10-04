package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/report/internal/model"
)

type TaskDataSource struct {
	tasks    taskv1.TaskServiceClient
	statuses taskv1.TaskStatusServiceClient
	timeout  time.Duration
}

func NewTaskDataSource(
	tasks taskv1.TaskServiceClient,
	statuses taskv1.TaskStatusServiceClient,
	timeout time.Duration,
) *TaskDataSource {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &TaskDataSource{tasks: tasks, statuses: statuses, timeout: timeout}
}

func (s *TaskDataSource) FetchStatuses(ctx context.Context, profileID string) ([]model.Status, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	response, err := s.statuses.ListTaskStatuses(callCtx, &taskv1.ListTaskStatusesRequest{
		ProfileId:       profileID,
		IncludeArchived: true,
	})
	if err != nil {
		return nil, fmt.Errorf("list task statuses cho profile %s: %w", profileID, err)
	}

	out := make([]model.Status, 0, len(response.GetStatuses()))
	for _, status := range response.GetStatuses() {
		out = append(out, model.Status{
			ID:       status.GetId(),
			Name:     status.GetName(),
			Slug:     status.GetSlug(),
			Color:    status.GetColor(),
			Category: statusCategoryToModel(status.GetCategory()),
			Position: status.GetPosition(),
		})
	}
	return out, nil
}

func (s *TaskDataSource) FetchTasks(ctx context.Context, profileID string, includeArchived bool) ([]model.Task, error) {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	response, err := s.tasks.ListTasks(callCtx, &taskv1.ListTasksRequest{
		ProfileId:       profileID,
		IncludeArchived: includeArchived,
	})
	if err != nil {
		return nil, fmt.Errorf("list tasks cho profile %s: %w", profileID, err)
	}

	out := make([]model.Task, 0, len(response.GetTasks()))
	for _, task := range response.GetTasks() {
		out = append(out, model.Task{
			ID:          task.GetId(),
			ParentID:    task.GetParentTaskId(),
			StatusID:    task.GetStatusId(),
			Title:       task.GetTitle(),
			Priority:    priorityToModel(task.GetPriority()),
			DueAt:       timePtr(task.GetDueAt()),
			CompletedAt: timePtr(task.GetCompletedAt()),
			CreatedAt:   timeValue(task.GetCreatedAt()),
			IsArchived:  task.GetIsArchived(),
		})
	}
	return out, nil
}

func statusCategoryToModel(category taskv1.TaskStatusCategory) model.StatusCategory {
	switch category {
	case taskv1.TaskStatusCategory_TASK_STATUS_CATEGORY_IN_PROGRESS:
		return model.CategoryInProgress
	case taskv1.TaskStatusCategory_TASK_STATUS_CATEGORY_DONE:
		return model.CategoryDone
	case taskv1.TaskStatusCategory_TASK_STATUS_CATEGORY_CANCELLED:
		return model.CategoryCancelled
	default:
		return model.CategoryTodo
	}
}

func priorityToModel(priority taskv1.TaskPriority) string {
	name := priority.String()
	return strings.ToLower(strings.TrimPrefix(name, "TASK_PRIORITY_"))
}

func timePtr(value *timestamppb.Timestamp) *time.Time {
	if value == nil {
		return nil
	}
	converted := value.AsTime().UTC()
	return &converted
}

func timeValue(value *timestamppb.Timestamp) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.AsTime().UTC()
}
