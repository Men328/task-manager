package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"taskmanager/service/worker/internal/dependency"
	"taskmanager/service/worker/internal/model"
	"taskmanager/service/worker/internal/service"
)

type KafkaReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
}

type RetryPolicy struct {
	Initial        time.Duration
	Max            time.Duration
	PersistTimeout time.Duration
}

type MessageConsumer struct {
	reader    KafkaReader
	persister service.MessagePersister
	retry     RetryPolicy
	sleep     func(ctx context.Context, d time.Duration) error
}

func NewMessageConsumer(reader KafkaReader, persister service.MessagePersister, retry RetryPolicy) *MessageConsumer {
	return &MessageConsumer{reader: reader, persister: persister, retry: retry, sleep: sleepContext}
}

func (c *MessageConsumer) Run(ctx context.Context) error {
	for {
		record, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("đọc kafka thất bại", "error", err)
			if err := c.sleep(ctx, c.retry.Initial); err != nil {
				return nil
			}
			continue
		}

		if err := c.handle(ctx, record); err != nil {
			return nil
		}
		if err := c.commit(ctx, record); err != nil {
			return nil
		}
	}
}

func (c *MessageConsumer) handle(ctx context.Context, record kafka.Message) error {
	message, err := dependency.DecodeMessageEvent(record.Value)
	if err != nil {
		logSkipped(record, err)
		return nil
	}

	backoff := c.retry.Initial
	for attempt := 1; ; attempt++ {
		err := c.persist(ctx, message)
		if err == nil {
			slog.Debug("đã lưu tin nhắn", "message_id", message.ID, "partition", record.Partition, "offset", record.Offset)
			return nil
		}
		if errors.Is(err, model.ErrPermanent) {
			logSkipped(record, err)
			return nil
		}

		slog.Warn("lưu tin nhắn thất bại, thử lại",
			"message_id", message.ID, "attempt", attempt, "backoff", backoff.String(), "error", err)
		if err := c.sleep(ctx, backoff); err != nil {
			return err
		}
		backoff = min(backoff*2, c.retry.Max)
	}
}

func (c *MessageConsumer) persist(ctx context.Context, message model.Message) error {
	callCtx, cancel := context.WithTimeout(ctx, c.retry.PersistTimeout)
	defer cancel()
	return c.persister.Persist(callCtx, message)
}

func (c *MessageConsumer) commit(ctx context.Context, record kafka.Message) error {
	backoff := c.retry.Initial
	for {
		err := c.reader.CommitMessages(ctx, record)
		if err == nil {
			return nil
		}
		slog.Warn("commit offset thất bại, thử lại", "partition", record.Partition, "offset", record.Offset, "error", err)
		if err := c.sleep(ctx, backoff); err != nil {
			return err
		}
		backoff = min(backoff*2, c.retry.Max)
	}
}

func logSkipped(record kafka.Message, err error) {
	slog.Error("bỏ qua sự kiện không xử lý được",
		"topic", record.Topic,
		"partition", record.Partition,
		"offset", record.Offset,
		"payload", string(record.Value),
		"error", err,
	)
}

func sleepContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
