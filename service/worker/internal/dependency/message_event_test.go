package dependency

import (
	"errors"
	"testing"
	"time"

	"taskmanager/service/worker/internal/model"
)

func TestDecodeMessageEvent(t *testing.T) {
	raw := []byte(`{"id":"m1","conversationId":"c1","senderId":"a","recipientId":"b","body":"chào","clientId":"x","createdAt":"2026-11-08T09:30:00.123456Z","futureField":1}`)

	message, err := DecodeMessageEvent(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	want := time.Date(2026, 11, 8, 9, 30, 0, 123456000, time.UTC)
	if message.ID != "m1" || message.ConversationID != "c1" || message.Body != "chào" || !message.CreatedAt.Equal(want) {
		t.Fatalf("decode sai: %+v", message)
	}
}

func TestDecodeMessageEventRejectsGarbageAsPermanent(t *testing.T) {
	if _, err := DecodeMessageEvent([]byte("not-json")); !errors.Is(err, model.ErrPermanent) {
		t.Fatalf("payload hỏng phải là lỗi vĩnh viễn, nhận %v", err)
	}
}

func TestDecodeMessageEventWithTask(t *testing.T) {
	raw := []byte(`{"id":"m1","conversationId":"c1","senderId":"a","recipientId":"b","body":"Mua sữa","createdAt":"2026-11-08T09:30:00Z","task":{"taskId":"t1","title":"Mua sữa","statusName":"To do","statusCategory":"todo"}}`)

	message, err := DecodeMessageEvent(raw)
	if err != nil || message.Task == nil || message.Task.TaskID != "t1" || message.Task.StatusCategory != "todo" {
		t.Fatalf("phải đọc được thẻ task: %+v %v", message.Task, err)
	}
}

func TestDecodeTaskStatusChanged(t *testing.T) {
	raw := []byte(`{"eventId":"e1","taskId":"t1","profileId":"bob","title":"Mua sữa","requestedBy":"alice","sourceMessageId":"m1",
		"fromStatus":{"name":"To do","category":"TASK_STATUS_CATEGORY_TODO"},
		"toStatus":{"name":"Done","color":"#0a0","category":"TASK_STATUS_CATEGORY_DONE"},
		"changedAt":"2026-11-08T10:00:00Z"}`)

	event, err := DecodeTaskStatusChanged(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if event.OwnerID != "bob" || event.RequestedBy != "alice" || event.MessageID != "m1" ||
		event.FromStatusName != "To do" || event.ToStatusName != "Done" || event.ToCategory != "done" || event.ToColor != "#0a0" {
		t.Fatalf("decode sai: %+v", event)
	}
	if _, err := DecodeTaskStatusChanged([]byte("{")); !errors.Is(err, model.ErrPermanent) {
		t.Fatalf("payload hỏng phải là lỗi vĩnh viễn, nhận %v", err)
	}
}
