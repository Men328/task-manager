package repository

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/worker/internal/model"
)

const taskStatusChangedNoticeType = "TASK_STATUS_CHANGED"

type NotificationClient struct {
	client  notificationv1.NoticeServiceClient
	timeout time.Duration
}

func NewNotificationClient(client notificationv1.NoticeServiceClient, timeout time.Duration) *NotificationClient {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &NotificationClient{client: client, timeout: timeout}
}

func (c *NotificationClient) TaskStatusChanged(ctx context.Context, event model.TaskStatusChanged) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.CreateNotice(callCtx, &notificationv1.CreateNoticeRequest{
		ProfileId:  event.RequestedBy,
		Type:       taskStatusChangedNoticeType,
		Title:      TaskNoticeTitle(event),
		Body:       TaskNoticeBody(event),
		TargetType: notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_MESSAGE,
		TargetId:   event.MessageID,
		Source:     event.EventID,
	})
	if err == nil {
		return nil
	}
	if status.Code(err) == codes.InvalidArgument {
		return fmt.Errorf("%w: notification từ chối notice: %v", model.ErrPermanent, err)
	}
	return fmt.Errorf("tạo notice đổi trạng thái task: %w", err)
}

func TaskNoticeTitle(event model.TaskStatusChanged) string {
	switch event.ToCategory {
	case "done":
		return fmt.Sprintf("Task “%s” đã hoàn thành", event.Title)
	case "cancelled":
		return fmt.Sprintf("Task “%s” đã bị huỷ", event.Title)
	default:
		return fmt.Sprintf("Task “%s” chuyển sang %s", event.Title, event.ToStatusName)
	}
}

func TaskNoticeBody(event model.TaskStatusChanged) string {
	if event.FromStatusName == "" {
		return "Trạng thái mới: " + event.ToStatusName
	}
	return event.FromStatusName + " → " + event.ToStatusName
}
