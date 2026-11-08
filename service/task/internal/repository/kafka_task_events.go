package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/task/internal/model"
)

type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type KafkaTaskEvents struct {
	writer KafkaWriter
}

func NewKafkaTaskEventWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		Async:                  true,
		BatchTimeout:           10 * time.Millisecond,
		MaxAttempts:            10,
		AllowAutoTopicCreation: false,
		Completion:             logFailedTaskEvents,
	}
}

func NewKafkaTaskEvents(writer KafkaWriter) *KafkaTaskEvents {
	return &KafkaTaskEvents{writer: writer}
}

func (p *KafkaTaskEvents) PublishStatusChanged(ctx context.Context, event model.StatusChangedEvent) error {
	value, err := EncodeStatusChangedEvent(event)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(event.Task.ID),
		Value:   value,
		Headers: []kafka.Header{{Key: "event-type", Value: []byte("task.status_changed")}},
		Time:    event.ChangedAt,
	})
}

func EncodeStatusChangedEvent(event model.StatusChangedEvent) ([]byte, error) {
	value, err := protojson.Marshal(&taskv1.TaskStatusChangedEvent{
		EventId:         uuid.NewString(),
		TaskId:          event.Task.ID,
		ProfileId:       event.Task.ProfileID,
		Title:           event.Task.Title,
		RequestedBy:     event.Task.RequestedBy,
		SourceMessageId: event.Task.SourceMessageID,
		FromStatus:      statusEventProto(event.From),
		ToStatus:        statusEventProto(event.To),
		ChangedAt:       timestamppb.New(event.ChangedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("mã hoá sự kiện task: %w", err)
	}
	return value, nil
}

func statusEventProto(s model.Status) *taskv1.TaskStatus {
	return &taskv1.TaskStatus{
		Id:         s.ID,
		ProfileId:  s.ProfileID,
		Name:       s.Name,
		Slug:       s.Slug,
		Color:      s.Color,
		Category:   taskv1.TaskStatusCategory(s.Category),
		IsTerminal: s.IsTerminal,
	}
}

func logFailedTaskEvents(messages []kafka.Message, err error) {
	if err == nil {
		return
	}
	for _, m := range messages {
		slog.Error("kafka không nhận sự kiện task sau khi thử lại",
			"topic", m.Topic, "task_id", string(m.Key), "payload", string(m.Value), "error", err)
	}
}
