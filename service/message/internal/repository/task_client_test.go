package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/message/internal/model"
)

type stubTaskClient struct {
	taskv1.TaskServiceClient
	request *taskv1.CreateTaskFromMessageRequest
	err     error
}

func (s *stubTaskClient) CreateTaskFromMessage(_ context.Context, in *taskv1.CreateTaskFromMessageRequest, _ ...grpc.CallOption) (*taskv1.CreateTaskFromMessageResponse, error) {
	s.request = in
	if s.err != nil {
		return nil, s.err
	}
	return &taskv1.CreateTaskFromMessageResponse{
		Task:   &taskv1.Task{Id: "t1", Title: in.GetTitle()},
		Status: &taskv1.TaskStatus{Name: "Đang làm", Color: "#123456", Category: taskv1.TaskStatusCategory_TASK_STATUS_CATEGORY_IN_PROGRESS},
	}, nil
}

func TestTaskClientMapsRequestAndStatus(t *testing.T) {
	stub := &stubTaskClient{}
	due := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	ref, err := NewTaskClient(stub, time.Second).CreateForMessage(context.Background(), model.TaskRequest{
		RecipientID: "bob", RequestedBy: "alice", MessageID: "m1",
		Draft: model.TaskDraft{Title: "Mua sữa", Priority: "urgent", DueAt: &due},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	in := stub.request
	if in.GetProfileId() != "bob" || in.GetRequestedBy() != "alice" || in.GetSourceMessageId() != "m1" ||
		in.GetPriority() != taskv1.TaskPriority_TASK_PRIORITY_URGENT || !in.GetDueAt().AsTime().Equal(due) || in.GetProjectId() != "" {
		t.Fatalf("request sai: %+v", in)
	}
	if ref.TaskID != "t1" || ref.StatusName != "Đang làm" || ref.StatusCategory != "in_progress" || ref.StatusColor != "#123456" {
		t.Fatalf("ref sai: %+v", ref)
	}
}

func TestTaskClientMapsUnavailable(t *testing.T) {
	stub := &stubTaskClient{err: status.Error(codes.Unavailable, "down")}
	if _, err := NewTaskClient(stub, time.Second).CreateForMessage(context.Background(), model.TaskRequest{}); !errors.Is(err, model.ErrUnavailable) {
		t.Fatalf("phải ra ErrUnavailable, nhận %v", err)
	}
}

func TestStatusCategoryName(t *testing.T) {
	cases := map[taskv1.TaskStatusCategory]string{
		taskv1.TaskStatusCategory_TASK_STATUS_CATEGORY_TODO:        "todo",
		taskv1.TaskStatusCategory_TASK_STATUS_CATEGORY_DONE:        "done",
		taskv1.TaskStatusCategory_TASK_STATUS_CATEGORY_CANCELLED:   "cancelled",
		taskv1.TaskStatusCategory_TASK_STATUS_CATEGORY_UNSPECIFIED: "",
	}
	for in, want := range cases {
		if got := StatusCategoryName(in); got != want {
			t.Fatalf("%v: phải %q, nhận %q", in, want, got)
		}
	}
}
