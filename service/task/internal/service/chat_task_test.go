package service

import (
	"context"
	"errors"
	"testing"

	"taskmanager/service/task/internal/model"
)

type fakeEvents struct {
	events []model.StatusChangedEvent
	err    error
}

func (f *fakeEvents) PublishStatusChanged(_ context.Context, event model.StatusChangedEvent) error {
	f.events = append(f.events, event)
	return f.err
}

func newChatTaskService(repo *fakeTaskRepo, checker *fakeProjectChecker, events *fakeEvents) TaskService {
	return NewTaskService(repo, fakeTaskStatusRepo{}, fakeTaskTransitionRepo{}, checker, events)
}

func TestCreateFromMessageUsesRecipientDefaultProjectAndDefaultStatus(t *testing.T) {
	repo := newFakeTaskRepo()
	checker := &fakeProjectChecker{}
	svc := newChatTaskService(repo, checker, &fakeEvents{})

	created, status, err := svc.CreateFromMessage(context.Background(), model.Task{
		ProfileID: "u1", Title: "Mua sữa", RequestedBy: "alice", SourceMessageID: "m1",
		StatusID: "st-done", ParentTaskID: "x", RecurrenceRule: "FREQ=DAILY",
	})
	if err != nil {
		t.Fatalf("CreateFromMessage: %v", err)
	}
	if created.ProjectID != "default-u1" || len(checker.calls) != 1 || checker.calls[0] != "default-u1" {
		t.Fatalf("phải vào project mặc định của người nhận, nhận %q (calls %v)", created.ProjectID, checker.calls)
	}
	if created.StatusID != "st-default" || status.ID != "st-default" {
		t.Fatalf("phải dùng status mặc định, nhận task=%s status=%s", created.StatusID, status.ID)
	}
	if created.ParentTaskID != "" || created.RecurrenceRule != "" {
		t.Fatal("task giao qua chat không nhận parent/lặp từ client")
	}
	if created.RequestedBy != "alice" || created.SourceMessageID != "m1" {
		t.Fatalf("phải nhớ người giao + tin nhắn: %+v", created)
	}
}

func TestCreateFromMessageKeepsExplicitProject(t *testing.T) {
	checker := &fakeProjectChecker{}
	created, _, err := newChatTaskService(newFakeTaskRepo(), checker, &fakeEvents{}).CreateFromMessage(context.Background(),
		model.Task{ProfileID: "u1", ProjectID: "p9", Title: "x", RequestedBy: "alice", SourceMessageID: "m1"})
	if err != nil || created.ProjectID != "p9" {
		t.Fatalf("project chỉ định phải giữ nguyên: %+v %v", created, err)
	}
}

func TestCreateFromMessageRejectsMissingRequesterOrSelf(t *testing.T) {
	svc := newChatTaskService(newFakeTaskRepo(), &fakeProjectChecker{}, &fakeEvents{})
	cases := []model.Task{
		{ProfileID: "u1", Title: "x", SourceMessageID: "m1"},
		{ProfileID: "u1", Title: "x", RequestedBy: "alice"},
		{ProfileID: "u1", Title: "x", RequestedBy: "u1", SourceMessageID: "m1"},
	}
	for _, tc := range cases {
		if _, _, err := svc.CreateFromMessage(context.Background(), tc); !isKind(err, model.ErrorKindInvalidArgument) {
			t.Fatalf("%+v: phải invalid_argument, nhận %v", tc, err)
		}
	}
}

func TestChangeStatusPublishesOnlyForChatTasks(t *testing.T) {
	repo := newFakeTaskRepo()
	events := &fakeEvents{}
	svc := newChatTaskService(repo, &fakeProjectChecker{}, events)
	repo.put(model.Task{ID: "chat", ProfileID: "u1", ProjectID: "p1", StatusID: "st-default", Title: "Mua sữa", RequestedBy: "alice", SourceMessageID: "m1"})
	repo.put(model.Task{ID: "own", ProfileID: "u1", ProjectID: "p1", StatusID: "st-default", Title: "Việc riêng"})

	if _, err := svc.ChangeStatus(context.Background(), "own", "st-done", ""); err != nil {
		t.Fatalf("own: %v", err)
	}
	if len(events.events) != 0 {
		t.Fatal("task tự tạo không phát sự kiện")
	}

	if _, err := svc.ChangeStatus(context.Background(), "chat", "st-done", ""); err != nil {
		t.Fatalf("chat: %v", err)
	}
	if len(events.events) != 1 {
		t.Fatalf("phải phát 1 sự kiện, nhận %d", len(events.events))
	}
	got := events.events[0]
	if got.Task.RequestedBy != "alice" || got.From.ID != "st-default" || got.To.ID != "st-done" || got.ChangedAt.IsZero() {
		t.Fatalf("sự kiện sai: %+v", got)
	}
}

func TestChangeStatusSucceedsWhenPublishFails(t *testing.T) {
	repo := newFakeTaskRepo()
	svc := newChatTaskService(repo, &fakeProjectChecker{}, &fakeEvents{err: errors.New("kafka down")})
	repo.put(model.Task{ID: "chat", ProfileID: "u1", ProjectID: "p1", StatusID: "st-default", Title: "x", RequestedBy: "alice", SourceMessageID: "m1"})

	change, err := svc.ChangeStatus(context.Background(), "chat", "st-done", "")
	if err != nil || change.Task.StatusID != "st-done" {
		t.Fatalf("lỗi kafka không được làm hỏng việc đổi trạng thái: %+v %v", change, err)
	}
}
