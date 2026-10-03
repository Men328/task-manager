package repository

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	calendarv1 "taskmanager/common/gen/go/calendar/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type scheduleClient struct {
	client  calendarv1.CalendarServiceClient
	timeout time.Duration
}

func NewScheduleClient(client calendarv1.CalendarServiceClient, timeout time.Duration) *scheduleClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &scheduleClient{client: client, timeout: timeout}
}

func (c *scheduleClient) Create(ctx context.Context, in model.ScheduleInput) (model.ScheduleRef, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	request := &calendarv1.CreateScheduleRequest{
		ProfileId:   in.ProfileID,
		Title:       in.Title,
		Description: in.Description,
		Location:    in.Location,
		StartAt:     timestamppb.New(in.StartAt),
		AllDay:      in.AllDay,
	}
	if in.EndAt != nil {
		request.EndAt = timestamppb.New(*in.EndAt)
	}

	response, err := c.client.CreateSchedule(callCtx, request)
	if err != nil {
		return model.ScheduleRef{}, fmt.Errorf("%w: %v", model.ErrScheduleCreate, err)
	}

	schedule := response.GetSchedule()
	return model.ScheduleRef{
		ID:    schedule.GetId(),
		Title: schedule.GetTitle(),
	}, nil
}
