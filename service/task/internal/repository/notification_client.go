package repository

import (
	"context"
	"fmt"
	"time"

	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/task/internal/model"
)

const taskReminderNoticeType = "TASK_REMINDER"

type NotificationClient struct {
	client   notificationv1.NoticeServiceClient
	timeout  time.Duration
	location *time.Location
}

func NewNotificationClient(client notificationv1.NoticeServiceClient, timeout time.Duration, location *time.Location) *NotificationClient {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	if location == nil {
		location = time.UTC
	}
	return &NotificationClient{client: client, timeout: timeout, location: location}
}

func (c *NotificationClient) SendTaskReminder(ctx context.Context, t model.Task) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	_, err := c.client.CreateNotice(callCtx, &notificationv1.CreateNoticeRequest{
		ProfileId:  t.ProfileID,
		Type:       taskReminderNoticeType,
		Title:      "Nhắc việc: " + t.Title,
		Body:       c.reminderBody(t),
		TargetType: notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_TASK,
		TargetId:   t.ID,
	})
	if err != nil {
		return fmt.Errorf("create reminder notice: %w", err)
	}
	return nil
}

func (c *NotificationClient) reminderBody(t model.Task) string {
	if t.DueAt == nil {
		return ""
	}
	due := t.DueAt.In(c.location)
	if due.Hour() == 0 && due.Minute() == 0 {
		return "Hạn: " + due.Format("02/01/2006")
	}
	return "Hạn: " + due.Format("15:04 02/01/2006")
}
