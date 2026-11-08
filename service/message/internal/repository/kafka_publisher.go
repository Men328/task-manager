package repository

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	messagev1 "taskmanager/common/gen/go/message/v1"
	"taskmanager/service/message/internal/model"
)

const ContentTypeHeader = "content-type"

type KafkaWriter interface {
	WriteMessages(ctx context.Context, msgs ...kafka.Message) error
}

type KafkaPublisher struct {
	writer KafkaWriter
}

func NewKafkaWriter(brokers []string, topic string) *kafka.Writer {
	return &kafka.Writer{
		Addr:                   kafka.TCP(brokers...),
		Topic:                  topic,
		Balancer:               &kafka.Hash{},
		RequiredAcks:           kafka.RequireAll,
		Async:                  true,
		BatchTimeout:           10 * time.Millisecond,
		MaxAttempts:            10,
		AllowAutoTopicCreation: false,
		Completion:             logFailedDeliveries,
	}
}

func NewKafkaPublisher(writer KafkaWriter) *KafkaPublisher {
	return &KafkaPublisher{writer: writer}
}

func (p *KafkaPublisher) Publish(ctx context.Context, message model.Message) error {
	value, err := EncodeMessageEvent(message)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:     []byte(message.ConversationID),
		Value:   value,
		Headers: []kafka.Header{{Key: ContentTypeHeader, Value: []byte("application/json")}},
		Time:    message.CreatedAt,
	})
}

func EncodeMessageEvent(message model.Message) ([]byte, error) {
	value, err := protojson.Marshal(&messagev1.ChatMessage{
		Id:             message.ID,
		ConversationId: message.ConversationID,
		SenderId:       message.SenderID,
		RecipientId:    message.RecipientID,
		Body:           message.Body,
		ClientId:       message.ClientID,
		CreatedAt:      timestamppb.New(message.CreatedAt),
	})
	if err != nil {
		return nil, fmt.Errorf("mã hoá sự kiện kafka: %w", err)
	}
	return value, nil
}

func logFailedDeliveries(messages []kafka.Message, err error) {
	if err == nil {
		return
	}
	for _, m := range messages {
		slog.Error("kafka không nhận tin nhắn sau khi thử lại, tin nhắn sẽ không được lưu",
			"topic", m.Topic,
			"conversation_id", string(m.Key),
			"payload", string(m.Value),
			"error", err,
		)
	}
}
