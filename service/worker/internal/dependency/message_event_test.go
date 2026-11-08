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
