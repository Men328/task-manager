package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/message/internal/model"
)

type TaskClient struct {
	client  taskv1.TaskServiceClient
	timeout time.Duration
}

func NewTaskClient(client taskv1.TaskServiceClient, timeout time.Duration) *TaskClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &TaskClient{client: client, timeout: timeout}
}

func (c *TaskClient) CreateForMessage(ctx context.Context, request model.TaskRequest) (model.TaskRef, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	in := &taskv1.CreateTaskFromMessageRequest{
		ProfileId:       request.RecipientID,
		Title:           request.Draft.Title,
		Description:     request.Draft.Description,
		Priority:        priorityToProto(request.Draft.Priority),
		RequestedBy:     request.RequestedBy,
		SourceMessageId: request.MessageID,
	}
	if request.Draft.DueAt != nil {
		in.DueAt = timestamppb.New(*request.Draft.DueAt)
	}

	response, err := c.client.CreateTaskFromMessage(callCtx, in)
	if err != nil {
		switch status.Code(err) {
		case codes.Unavailable, codes.DeadlineExceeded, codes.Canceled:
			return model.TaskRef{}, fmt.Errorf("%w: task CreateTaskFromMessage: %v", model.ErrUnavailable, err)
		default:
			return model.TaskRef{}, err
		}
	}

	task := response.GetTask()
	st := response.GetStatus()
	return model.TaskRef{
		TaskID:         task.GetId(),
		Title:          task.GetTitle(),
		StatusName:     st.GetName(),
		StatusCategory: StatusCategoryName(st.GetCategory()),
		StatusColor:    st.GetColor(),
	}, nil
}

func priorityToProto(priority string) taskv1.TaskPriority {
	switch priority {
	case "low":
		return taskv1.TaskPriority_TASK_PRIORITY_LOW
	case "high":
		return taskv1.TaskPriority_TASK_PRIORITY_HIGH
	case "urgent":
		return taskv1.TaskPriority_TASK_PRIORITY_URGENT
	default:
		return taskv1.TaskPriority_TASK_PRIORITY_MEDIUM
	}
}

func StatusCategoryName(category taskv1.TaskStatusCategory) string {
	name := strings.TrimPrefix(category.String(), "TASK_STATUS_CATEGORY_")
	if name == "UNSPECIFIED" || name == category.String() {
		return ""
	}
	return strings.ToLower(name)
}
