package repository

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	eventv1 "taskmanager/common/gen/go/event/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type eventClient struct {
	client  eventv1.EventServiceClient
	timeout time.Duration
}

func NewEventClient(client eventv1.EventServiceClient, timeout time.Duration) *eventClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &eventClient{client: client, timeout: timeout}
}

func (c *eventClient) Create(ctx context.Context, in model.EventInput) (model.EventRef, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	request := &eventv1.CreateEventRequest{
		ProfileId:   in.ProfileID,
		Title:       in.Title,
		Description: in.Description,
		Location:    in.Location,
		StartAt:     timestamppb.New(in.StartAt),
		AllDay:      in.AllDay,
		Status:      eventv1.EventStatus_EVENT_STATUS_PLANNED,
		Source:      in.Source,
	}
	if in.EndAt != nil {
		request.EndAt = timestamppb.New(*in.EndAt)
	}

	response, err := c.client.CreateEvent(callCtx, request)
	if err != nil {
		return model.EventRef{}, fmt.Errorf("%w: %v", model.ErrEventCreate, err)
	}

	event := response.GetEvent()
	return model.EventRef{
		ID:    event.GetId(),
		Title: event.GetTitle(),
	}, nil
}
