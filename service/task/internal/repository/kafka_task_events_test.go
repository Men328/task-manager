package repository

import (
	"context"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/model"
)

type captureWriter struct {
	messages []kafka.Message
}

func (w *captureWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	w.messages = append(w.messages, msgs...)
	return nil
}

func TestKafkaTaskEventsKeysByTaskAndEncodesStatuses(t *testing.T) {
	writer := &captureWriter{}
	changedAt := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
	event := model.StatusChangedEvent{
		Task:      model.Task{ID: "t1", ProfileID: "bob", Title: "Mua sữa", RequestedBy: "alice", SourceMessageID: "m1"},
		From:      model.Status{ID: "s1", Name: "To do", Category: model.TaskStatusCategoryTodo},
		To:        model.Status{ID: "s2", Name: "Done", Color: "#00aa00", Category: model.TaskStatusCategoryDone},
		ChangedAt: changedAt,
	}

	if err := NewKafkaTaskEvents(writer).PublishStatusChanged(context.Background(), event); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if len(writer.messages) != 1 || string(writer.messages[0].Key) != "t1" {
		t.Fatalf("key phải là task id, nhận %+v", writer.messages)
	}

	var decoded taskv1.TaskStatusChangedEvent
	if err := protojson.Unmarshal(writer.messages[0].Value, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.GetRequestedBy() != "alice" || decoded.GetSourceMessageId() != "m1" || decoded.GetEventId() == "" {
		t.Fatalf("thiếu thông tin người giao/tin nhắn: %+v", &decoded)
	}
	if decoded.GetToStatus().GetCategory() != taskv1.TaskStatusCategory_TASK_STATUS_CATEGORY_DONE || decoded.GetToStatus().GetName() != "Done" {
		t.Fatalf("status đích sai: %+v", decoded.GetToStatus())
	}
	if !decoded.GetChangedAt().AsTime().Equal(changedAt) {
		t.Fatal("changed_at sai")
	}
}
