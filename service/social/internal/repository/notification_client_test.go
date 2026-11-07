package repository

import (
	"context"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"

	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/social/internal/model"
)

type captureNoticeClient struct {
	notificationv1.NoticeServiceClient
	got *notificationv1.CreateNoticeRequest
}

func (c *captureNoticeClient) CreateNotice(_ context.Context, in *notificationv1.CreateNoticeRequest, _ ...grpc.CallOption) (*notificationv1.CreateNoticeResponse, error) {
	c.got = in
	return &notificationv1.CreateNoticeResponse{}, nil
}

func TestNotificationClientFriendRequested(t *testing.T) {
	capture := &captureNoticeClient{}
	client := NewNotificationClient(capture, time.Second)

	if err := client.FriendRequested(context.Background(), "to", model.Profile{DisplayName: "An"}, "req"); err != nil {
		t.Fatalf("gửi: %v", err)
	}
	assertFriendNotice(t, capture.got, "FRIEND_REQUEST")
}

func TestNotificationClientFriendAccepted(t *testing.T) {
	capture := &captureNoticeClient{}
	client := NewNotificationClient(capture, time.Second)

	if err := client.FriendAccepted(context.Background(), "to", model.Profile{DisplayName: "An"}, "req"); err != nil {
		t.Fatalf("gửi: %v", err)
	}
	assertFriendNotice(t, capture.got, "FRIEND_ACCEPTED")
}

func assertFriendNotice(t *testing.T, got *notificationv1.CreateNoticeRequest, noticeType string) {
	t.Helper()
	if got.GetType() != noticeType || got.GetProfileId() != "to" || got.GetTargetId() != "req" {
		t.Fatalf("notice sai: %+v", got)
	}
	if got.GetTargetType() != notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_FRIEND {
		t.Fatalf("loại đích phải là FRIEND, nhận %v", got.GetTargetType())
	}
	if !strings.Contains(got.GetTitle(), "An") {
		t.Fatalf("title phải có tên, nhận %q", got.GetTitle())
	}
}
