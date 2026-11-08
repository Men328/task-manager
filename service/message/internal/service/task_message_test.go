package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"taskmanager/service/message/internal/model"
)

func TestSendTaskCreatesTaskForPeerAndDeliversCard(t *testing.T) {
	f := newFixture()
	due := time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)

	message, err := f.svc.SendTask(context.Background(), alice, bob, model.TaskDraft{Title: "  Mua sữa  ", Description: "2 hộp", DueAt: &due}, "c-1")
	if err != nil {
		t.Fatalf("SendTask: %v", err)
	}
	if len(f.tasks.requests) != 1 {
		t.Fatalf("phải tạo đúng 1 task, nhận %d", len(f.tasks.requests))
	}
	request := f.tasks.requests[0]
	if request.RecipientID != bob || request.RequestedBy != alice || request.MessageID != message.ID {
		t.Fatalf("task phải thuộc người nhận, nhớ người giao + id tin nhắn: %+v", request)
	}
	if request.Draft.Title != "Mua sữa" || request.Draft.Priority != "medium" || !request.Draft.DueAt.Equal(due) {
		t.Fatalf("draft phải được chuẩn hoá: %+v", request.Draft)
	}
	if message.Task == nil || message.Task.TaskID != "t1" || message.Body != "Mua sữa" {
		t.Fatalf("tin nhắn phải kèm thẻ task: %+v", message)
	}
	if len(f.broadcaster.sent) != 1 || f.broadcaster.sent[0].Task == nil || len(f.publisher.published) != 1 {
		t.Fatal("tin giao task phải đi cùng đường broadcast + kafka")
	}
}

func TestSendTaskValidatesBeforeCreatingTask(t *testing.T) {
	cases := map[string]struct {
		peer  string
		draft model.TaskDraft
		kind  model.ErrorKind
	}{
		"thiếu tiêu đề":     {bob, model.TaskDraft{Title: " "}, model.ErrorKindTaskTitleRequired},
		"tiêu đề quá dài":   {bob, model.TaskDraft{Title: strings.Repeat("a", model.MaxTaskTitleRunes+1)}, model.ErrorKindInvalidArgument},
		"priority lạ":       {bob, model.TaskDraft{Title: "x", Priority: "asap"}, model.ErrorKindInvalidArgument},
		"không phải bạn bè": {carol, model.TaskDraft{Title: "x"}, model.ErrorKindNotFriends},
		"tự giao mình":      {alice, model.TaskDraft{Title: "x"}, model.ErrorKindCannotMessageSelf},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture()
			_, err := f.svc.SendTask(context.Background(), alice, tc.peer, tc.draft, "")
			requireKind(t, err, tc.kind)
			if len(f.tasks.requests) != 0 || len(f.broadcaster.sent) != 0 {
				t.Fatal("bị từ chối thì không tạo task, không gửi tin")
			}
		})
	}
}

func TestSendTaskStopsWhenTaskCreationFails(t *testing.T) {
	f := newFixture()
	f.tasks.err = model.ErrUnavailable

	if _, err := f.svc.SendTask(context.Background(), alice, bob, model.TaskDraft{Title: "x"}, ""); !errors.Is(err, model.ErrUnavailable) {
		t.Fatalf("phải trả lỗi task service, nhận %v", err)
	}
	if len(f.broadcaster.sent) != 0 || len(f.publisher.published) != 0 {
		t.Fatal("task chưa tạo được thì không gửi tin")
	}
}

func TestGetMessageOnlyForParticipants(t *testing.T) {
	f := newFixture()
	id := "0193a5b0-0000-7000-8000-000000000001"
	f.reader.messages = []model.Message{{ID: id, SenderID: alice, RecipientID: bob}}

	for _, who := range []string{alice, bob} {
		if got, err := f.svc.Get(context.Background(), who, id); err != nil || got.ID != id {
			t.Fatalf("%s phải đọc được: %v", who, err)
		}
	}
	_, err := f.svc.Get(context.Background(), carol, id)
	requireKind(t, err, model.ErrorKindMessageNotFound)

	_, err = f.svc.Get(context.Background(), alice, "0193a5b0-0000-7000-8000-000000000099")
	requireKind(t, err, model.ErrorKindMessageNotFound)

	_, err = f.svc.Get(context.Background(), alice, "not-a-uuid")
	requireKind(t, err, model.ErrorKindInvalidArgument)
}
