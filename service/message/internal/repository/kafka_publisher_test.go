package repository

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"

	messagev1 "taskmanager/common/gen/go/message/v1"
	"taskmanager/service/message/internal/model"
)

type captureWriter struct {
	messages []kafka.Message
}

func (w *captureWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.messages = append(w.messages, msgs...)
	return nil
}

func TestKafkaPublisherKeysByConversationAndEncodesProtoJSON(t *testing.T) {
	writer := &captureWriter{}
	createdAt := time.Date(2026, 11, 8, 9, 30, 0, 0, time.UTC)
	message := model.Message{ID: "m1", ConversationID: "c1", SenderID: "a", RecipientID: "b", Body: "hi", ClientID: "x", CreatedAt: createdAt}

	if err := NewKafkaPublisher(writer).Publish(context.Background(), message); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(writer.messages) != 1 || string(writer.messages[0].Key) != "c1" {
		t.Fatalf("key phải là conversation id để giữ thứ tự trong 1 partition, nhận %+v", writer.messages)
	}

	var decoded messagev1.ChatMessage
	if err := protojson.Unmarshal(writer.messages[0].Value, &decoded); err != nil {
		t.Fatalf("value phải là protojson ChatMessage: %v", err)
	}
	if decoded.GetId() != "m1" || decoded.GetSenderId() != "a" || !decoded.GetCreatedAt().AsTime().Equal(createdAt) {
		t.Fatalf("payload sai: %+v", &decoded)
	}
}
