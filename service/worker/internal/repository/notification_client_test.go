package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/worker/internal/model"
)

type stubNoticeClient struct {
	notificationv1.NoticeServiceClient
	request *notificationv1.CreateNoticeRequest
	err     error
}

func (s *stubNoticeClient) CreateNotice(_ context.Context, in *notificationv1.CreateNoticeRequest, _ ...grpc.CallOption) (*notificationv1.CreateNoticeResponse, error) {
	s.request = in
	return &notificationv1.CreateNoticeResponse{}, s.err
}

func TestNotificationClientTargetsMessageOfRequester(t *testing.T) {
	stub := &stubNoticeClient{}
	event := model.TaskStatusChanged{EventID: "e1", RequestedBy: "alice", MessageID: "m1", Title: "Mua sữa", FromStatusName: "To do", ToStatusName: "Done", ToCategory: "done"}

	if err := NewNotificationClient(stub, time.Second).TaskStatusChanged(context.Background(), event); err != nil {
		t.Fatalf("notify: %v", err)
	}
	in := stub.request
	if in.GetProfileId() != "alice" || in.GetTargetType() != notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_MESSAGE || in.GetTargetId() != "m1" {
		t.Fatalf("notice phải gửi người giao và trỏ về tin nhắn: %+v", in)
	}
	if in.GetType() != "TASK_STATUS_CHANGED" || in.GetTitle() != "Task “Mua sữa” đã hoàn thành" || in.GetBody() != "To do → Done" {
		t.Fatalf("nội dung notice sai: %+v", in)
	}
}

func TestNoticeTitleByCategory(t *testing.T) {
	cases := map[string]string{
		"done":        "Task “X” đã hoàn thành",
		"cancelled":   "Task “X” đã bị huỷ",
		"in_progress": "Task “X” chuyển sang Đang làm",
	}
	for category, want := range cases {
		if got := TaskNoticeTitle(model.TaskStatusChanged{Title: "X", ToStatusName: "Đang làm", ToCategory: category}); got != want {
			t.Fatalf("%s: %q", category, got)
		}
	}
}

func TestNotificationClientErrorKinds(t *testing.T) {
	stub := &stubNoticeClient{err: status.Error(codes.InvalidArgument, "bad")}
	if err := NewNotificationClient(stub, time.Second).TaskStatusChanged(context.Background(), model.TaskStatusChanged{}); !errors.Is(err, model.ErrPermanent) {
		t.Fatalf("InvalidArgument là lỗi vĩnh viễn, nhận %v", err)
	}
	stub = &stubNoticeClient{err: status.Error(codes.Unavailable, "down")}
	if err := NewNotificationClient(stub, time.Second).TaskStatusChanged(context.Background(), model.TaskStatusChanged{}); err == nil || errors.Is(err, model.ErrPermanent) {
		t.Fatalf("Unavailable là lỗi tạm thời, nhận %v", err)
	}
}
