package repository

import (
	"context"
	"fmt"
	"time"

	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/social/internal/model"
)

const (
	friendRequestNoticeType  = "FRIEND_REQUEST"
	friendAcceptedNoticeType = "FRIEND_ACCEPTED"
)

type NotificationClient struct {
	client  notificationv1.NoticeServiceClient
	timeout time.Duration
}

func NewNotificationClient(client notificationv1.NoticeServiceClient, timeout time.Duration) *NotificationClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &NotificationClient{client: client, timeout: timeout}
}

func (c *NotificationClient) FriendRequested(ctx context.Context, to string, from model.Profile, requestID string) error {
	return c.send(ctx, to, friendRequestNoticeType, from.DisplayName+" muốn kết bạn với bạn", requestID)
}

func (c *NotificationClient) FriendAccepted(ctx context.Context, to string, by model.Profile, requestID string) error {
	return c.send(ctx, to, friendAcceptedNoticeType, by.DisplayName+" đã chấp nhận lời mời kết bạn", requestID)
}

func (c *NotificationClient) send(ctx context.Context, to, noticeType, title, requestID string) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.CreateNotice(callCtx, &notificationv1.CreateNoticeRequest{
		ProfileId:  to,
		Type:       noticeType,
		Title:      title,
		TargetType: notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_FRIEND,
		TargetId:   requestID,
	})
	if err != nil {
		return fmt.Errorf("create friend notice: %w", err)
	}
	return nil
}
