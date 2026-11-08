package handler

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/segmentio/kafka-go"

	"taskmanager/service/worker/internal/model"
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

type Processor interface {
	Process(ctx context.Context, value []byte) error
}

type Consumer struct {
	name      string
	reader    KafkaReader
	processor Processor
	retry     RetryPolicy
	sleep     func(ctx context.Context, d time.Duration) error
}

func NewConsumer(name string, reader KafkaReader, processor Processor, retry RetryPolicy) *Consumer {
	return &Consumer{name: name, reader: reader, processor: processor, retry: retry, sleep: sleepContext}
}

func (c *Consumer) Name() string { return c.name }

func (c *Consumer) Run(ctx context.Context) error {
	for {
		record, err := c.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("đọc kafka thất bại", "consumer", c.name, "error", err)
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

func (c *Consumer) handle(ctx context.Context, record kafka.Message) error {
	backoff := c.retry.Initial
	for attempt := 1; ; attempt++ {
		err := c.process(ctx, record)
		if err == nil {
			slog.Debug("đã xử lý sự kiện", "consumer", c.name, "partition", record.Partition, "offset", record.Offset)
			return nil
		}
		if errors.Is(err, model.ErrPermanent) {
			logSkipped(c.name, record, err)
			return nil
		}

		slog.Warn("xử lý sự kiện thất bại, thử lại",
			"consumer", c.name, "partition", record.Partition, "offset", record.Offset,
			"attempt", attempt, "backoff", backoff.String(), "error", err)
		if err := c.sleep(ctx, backoff); err != nil {
			return err
		}
		backoff = min(backoff*2, c.retry.Max)
	}
}

func (c *Consumer) process(ctx context.Context, record kafka.Message) error {
	callCtx, cancel := context.WithTimeout(ctx, c.retry.PersistTimeout)
	defer cancel()
	return c.processor.Process(callCtx, record.Value)
}

func (c *Consumer) commit(ctx context.Context, record kafka.Message) error {
	backoff := c.retry.Initial
	for {
		err := c.reader.CommitMessages(ctx, record)
		if err == nil {
			return nil
		}
		slog.Warn("commit offset thất bại, thử lại", "consumer", c.name, "partition", record.Partition, "offset", record.Offset, "error", err)
		if err := c.sleep(ctx, backoff); err != nil {
			return err
		}
		backoff = min(backoff*2, c.retry.Max)
	}
}

func logSkipped(name string, record kafka.Message, err error) {
	slog.Error("bỏ qua sự kiện không xử lý được",
		"consumer", name,
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
