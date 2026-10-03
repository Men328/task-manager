package repository

import (
	"context"
	"fmt"
	"time"

	backlogv1 "taskmanager/common/gen/go/backlog/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type backlogClient struct {
	client  backlogv1.BacklogServiceClient
	timeout time.Duration
}

func NewBacklogClient(client backlogv1.BacklogServiceClient, timeout time.Duration) *backlogClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &backlogClient{client: client, timeout: timeout}
}

func (c *backlogClient) Create(ctx context.Context, in model.BacklogInput) (model.BacklogRef, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.CreateBacklog(callCtx, &backlogv1.CreateBacklogRequest{
		ProfileId:   in.ProfileID,
		Title:       in.Title,
		Description: in.Description,
		Sender:      in.Sender,
		Source:      in.Source,
		Category:    in.Category,
		Reason:      in.Reason,
		ObjectKey:   in.ObjectKey,
	})
	if err != nil {
		return model.BacklogRef{}, fmt.Errorf("%w: %v", model.ErrBacklogCreate, err)
	}

	item := response.GetBacklog()
	return model.BacklogRef{
		ID:    item.GetId(),
		Title: item.GetTitle(),
	}, nil
}
