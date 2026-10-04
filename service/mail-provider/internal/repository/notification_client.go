package repository

import (
	"context"
	"fmt"
	"time"

	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type notificationClient struct {
	client  notificationv1.NoticeServiceClient
	timeout time.Duration
}

func NewNotificationClient(client notificationv1.NoticeServiceClient, timeout time.Duration) *notificationClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &notificationClient{client: client, timeout: timeout}
}

func (c *notificationClient) Notify(ctx context.Context, in model.NoticeInput) (model.NoticeRef, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.CreateNotice(callCtx, &notificationv1.CreateNoticeRequest{
		ProfileId:  in.ProfileID,
		Type:       in.Type,
		Title:      in.Title,
		Body:       in.Body,
		TargetType: noticeTargetTypeToProto(in.TargetType),
		TargetId:   in.TargetID,
		Source:     in.Source,
	})
	if err != nil {
		return model.NoticeRef{}, fmt.Errorf("%w: %v", model.ErrNoticeCreate, err)
	}

	notice := response.GetNotice()
	return model.NoticeRef{
		ID:    notice.GetId(),
		Title: notice.GetTitle(),
	}, nil
}

func noticeTargetTypeToProto(targetType string) notificationv1.NoticeTargetType {
	switch targetType {
	case model.NoticeTargetTask:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_TASK
	case model.NoticeTargetSchedule:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_SCHEDULE
	case model.NoticeTargetEvent:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_EVENT
	case model.NoticeTargetBacklog:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_BACKLOG
	default:
		return notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_UNSPECIFIED
	}
}
