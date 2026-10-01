package repository

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type taskClient struct {
	client  taskv1.TaskServiceClient
	timeout time.Duration
}

func NewTaskClient(client taskv1.TaskServiceClient, timeout time.Duration) *taskClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &taskClient{client: client, timeout: timeout}
}

func (c *taskClient) Create(ctx context.Context, in model.TaskInput) (model.TaskRef, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	request := &taskv1.CreateTaskRequest{
		ProfileId:   in.ProfileID,
		WorkspaceId: in.WorkspaceID,
		Title:       in.Title,
		Description: in.Description,
		Priority:    taskv1.TaskPriority(in.Priority),
	}
	if in.DueAt != nil {
		request.DueAt = timestamppb.New(*in.DueAt)
	}

	response, err := c.client.CreateTask(callCtx, request)
	if err != nil {
		return model.TaskRef{}, fmt.Errorf("%w: %v", model.ErrTaskCreate, err)
	}

	task := response.GetTask()
	return model.TaskRef{
		ID:          task.GetId(),
		Title:       task.GetTitle(),
		WorkspaceID: task.GetWorkspaceId(),
	}, nil
}
