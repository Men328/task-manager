package repository

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"

	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/task/internal/model"
)

type captureNoticeClient struct {
	notificationv1.NoticeServiceClient
	got *notificationv1.CreateNoticeRequest
}

func (c *captureNoticeClient) CreateNotice(_ context.Context, in *notificationv1.CreateNoticeRequest, _ ...grpc.CallOption) (*notificationv1.CreateNoticeResponse, error) {
	c.got = in
	return &notificationv1.CreateNoticeResponse{}, nil
}

func TestSendTaskReminderBuildsNotice(t *testing.T) {
	loc := time.FixedZone("ICT", 7*3600)
	capture := &captureNoticeClient{}
	client := NewNotificationClient(capture, time.Second, loc)

	dateOnly := time.Date(2026, 11, 2, 0, 0, 0, 0, loc)
	withTime := time.Date(2026, 11, 2, 9, 30, 0, 0, loc).UTC()
	cases := []struct {
		due  *time.Time
		body string
	}{
		{nil, ""},
		{&dateOnly, "Hạn: 02/11/2026"},
		{&withTime, "Hạn: 09:30 02/11/2026"},
	}
	for _, tc := range cases {
		task := model.Task{ID: "t1", ProfileID: "u1", Title: "Nộp báo cáo", DueAt: tc.due}
		if err := client.SendTaskReminder(context.Background(), task); err != nil {
			t.Fatalf("SendTaskReminder: %v", err)
		}
		got := capture.got
		if got.GetType() != "TASK_REMINDER" || got.GetTargetType() != notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_TASK ||
			got.GetTargetId() != "t1" || got.GetProfileId() != "u1" || got.GetTitle() != "Nhắc việc: Nộp báo cáo" {
			t.Fatalf("notice = %+v", got)
		}
		if got.GetBody() != tc.body {
			t.Fatalf("body = %q, want %q", got.GetBody(), tc.body)
		}
	}
}
