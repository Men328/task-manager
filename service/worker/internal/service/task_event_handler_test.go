package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/worker/internal/model"
	"taskmanager/service/worker/internal/service"
)

type fakeTaskStore struct {
	fakeStore
	updated bool
	message model.Message
	events  []model.TaskStatusChanged
	err     error
}

func (s *fakeTaskStore) UpdateTaskStatus(_ context.Context, event model.TaskStatusChanged) (model.Message, bool, error) {
	s.events = append(s.events, event)
	return s.message, s.updated, s.err
}

type fakeBroadcaster struct {
	sent []model.Message
	err  error
}

func (b *fakeBroadcaster) BroadcastUpdated(_ context.Context, message model.Message) error {
	b.sent = append(b.sent, message)
	return b.err
}

type fakeNotifier struct {
	sent []model.TaskStatusChanged
	err  error
}

func (n *fakeNotifier) TaskStatusChanged(_ context.Context, event model.TaskStatusChanged) error {
	n.sent = append(n.sent, event)
	return n.err
}

func taskEvent() model.TaskStatusChanged {
	return model.TaskStatusChanged{
		EventID: "e1", TaskID: "t1", OwnerID: "bob", RequestedBy: "alice", MessageID: "m1",
		Title: "Mua sữa", FromStatusName: "To do", ToStatusName: "Done", ToCategory: "done", ChangedAt: time.Now(),
	}
}

func TestTaskEventUpdatesCardBroadcastsAndNotifiesRequester(t *testing.T) {
	store := &fakeTaskStore{updated: true, message: model.Message{ID: "m1"}}
	broadcaster := &fakeBroadcaster{}
	notifier := &fakeNotifier{}

	if err := service.NewTaskEventHandler(store, broadcaster, notifier).Handle(context.Background(), taskEvent()); err != nil {
		t.Fatalf("handle: %v", err)
	}
	if len(store.events) != 1 || len(broadcaster.sent) != 1 || broadcaster.sent[0].ID != "m1" {
		t.Fatalf("phải cập nhật thẻ rồi broadcast, nhận store=%d broadcast=%d", len(store.events), len(broadcaster.sent))
	}
	if len(notifier.sent) != 1 || notifier.sent[0].RequestedBy != "alice" {
		t.Fatalf("phải báo người giao, nhận %+v", notifier.sent)
	}
}

func TestTaskEventStillNotifiesWhenCardNotUpdated(t *testing.T) {
	broadcaster := &fakeBroadcaster{}
	notifier := &fakeNotifier{}

	err := service.NewTaskEventHandler(&fakeTaskStore{}, broadcaster, notifier).Handle(context.Background(), taskEvent())
	if err != nil || len(broadcaster.sent) != 0 || len(notifier.sent) != 1 {
		t.Fatalf("thẻ chưa có vẫn phải báo, không broadcast: err=%v b=%d n=%d", err, len(broadcaster.sent), len(notifier.sent))
	}
}

func TestTaskEventBroadcastFailureDoesNotBlockNotice(t *testing.T) {
	notifier := &fakeNotifier{}
	store := &fakeTaskStore{updated: true}
	err := service.NewTaskEventHandler(store, &fakeBroadcaster{err: errors.New("soketi down")}, notifier).Handle(context.Background(), taskEvent())
	if err != nil || len(notifier.sent) != 1 {
		t.Fatalf("soketi lỗi không chặn notice: %v", err)
	}
}

func TestTaskEventNotifierErrorIsRetried(t *testing.T) {
	err := service.NewTaskEventHandler(&fakeTaskStore{}, &fakeBroadcaster{}, &fakeNotifier{err: errors.New("down")}).Handle(context.Background(), taskEvent())
	if err == nil || errors.Is(err, model.ErrPermanent) {
		t.Fatalf("lỗi notification là tạm thời để consumer thử lại, nhận %v", err)
	}
}

func TestTaskEventStoreErrorStopsBeforeNotice(t *testing.T) {
	notifier := &fakeNotifier{}
	err := service.NewTaskEventHandler(&fakeTaskStore{err: errors.New("db down")}, &fakeBroadcaster{}, notifier).Handle(context.Background(), taskEvent())
	if err == nil || len(notifier.sent) != 0 {
		t.Fatalf("DB lỗi thì dừng để thử lại, nhận %v", err)
	}
}

func TestTaskEventRejectsMalformedAsPermanent(t *testing.T) {
	for name, mutate := range map[string]func(*model.TaskStatusChanged){
		"thiếu người giao": func(e *model.TaskStatusChanged) { e.RequestedBy = "" },
		"thiếu tin nhắn":   func(e *model.TaskStatusChanged) { e.MessageID = "" },
		"thiếu thời gian":  func(e *model.TaskStatusChanged) { e.ChangedAt = time.Time{} },
	} {
		event := taskEvent()
		mutate(&event)
		err := service.NewTaskEventHandler(&fakeTaskStore{}, &fakeBroadcaster{}, &fakeNotifier{}).Handle(context.Background(), event)
		if !errors.Is(err, model.ErrPermanent) {
			t.Fatalf("%s: phải là lỗi vĩnh viễn, nhận %v", name, err)
		}
	}
}

func TestTaskEventSelfAssignedSkipsNotice(t *testing.T) {
	notifier := &fakeNotifier{}
	event := taskEvent()
	event.RequestedBy = event.OwnerID
	if err := service.NewTaskEventHandler(&fakeTaskStore{}, &fakeBroadcaster{}, notifier).Handle(context.Background(), event); err != nil || len(notifier.sent) != 0 {
		t.Fatalf("không tự báo cho chính chủ: %v", err)
	}
}
