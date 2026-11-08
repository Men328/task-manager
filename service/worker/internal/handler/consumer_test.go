package handler

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"

	"taskmanager/service/worker/internal/model"
)

type fakeReader struct {
	records   []kafka.Message
	committed []int64
	cancel    context.CancelFunc
}

func (r *fakeReader) FetchMessage(ctx context.Context) (kafka.Message, error) {
	if len(r.records) == 0 {
		r.cancel()
		<-ctx.Done()
		return kafka.Message{}, ctx.Err()
	}
	next := r.records[0]
	r.records = r.records[1:]
	return next, nil
}

func (r *fakeReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	for _, m := range msgs {
		r.committed = append(r.committed, m.Offset)
	}
	return nil
}

type fakePersister struct {
	failures int
	err      error
	saved    []string
	calls    int
}

func (p *fakePersister) Persist(_ context.Context, message model.Message) error {
	p.calls++
	if p.calls <= p.failures {
		return p.err
	}
	p.saved = append(p.saved, message.ID)
	return nil
}

func event(offset int64, id string) kafka.Message {
	value := fmt.Sprintf(`{"id":%q,"conversationId":"c","senderId":"a","recipientId":"b","body":"hi","createdAt":"2026-11-08T09:30:00Z"}`, id)
	return kafka.Message{Offset: offset, Value: []byte(value)}
}

func runConsumer(t *testing.T, reader *fakeReader, persister *fakePersister) []time.Duration {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	reader.cancel = cancel
	consumer := NewConsumer("chat", reader, NewChatMessageProcessor(persister), RetryPolicy{Initial: time.Millisecond, Max: 4 * time.Millisecond, PersistTimeout: time.Second})
	var sleeps []time.Duration
	consumer.sleep = func(ctx context.Context, d time.Duration) error {
		sleeps = append(sleeps, d)
		return ctx.Err()
	}

	done := make(chan error, 1)
	go func() { done <- consumer.Run(ctx) }()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("consumer không dừng khi context bị huỷ")
	}
	return sleeps
}

func TestConsumerPersistsThenCommitsInOrder(t *testing.T) {
	reader := &fakeReader{records: []kafka.Message{event(1, "m1"), event(2, "m2")}}
	persister := &fakePersister{}

	runConsumer(t, reader, persister)

	if fmt.Sprint(persister.saved) != "[m1 m2]" || fmt.Sprint(reader.committed) != "[1 2]" {
		t.Fatalf("phải lưu rồi commit theo thứ tự, nhận saved=%v committed=%v", persister.saved, reader.committed)
	}
}

func TestConsumerRetriesTransientErrorsWithBackoffBeforeCommit(t *testing.T) {
	reader := &fakeReader{records: []kafka.Message{event(7, "m1")}}
	persister := &fakePersister{failures: 4, err: errors.New("db down")}

	sleeps := runConsumer(t, reader, persister)

	if fmt.Sprint(persister.saved) != "[m1]" || fmt.Sprint(reader.committed) != "[7]" {
		t.Fatalf("phải lưu được sau khi thử lại, nhận saved=%v committed=%v", persister.saved, reader.committed)
	}
	if fmt.Sprint(sleeps) != "[1ms 2ms 4ms 4ms]" {
		t.Fatalf("backoff phải nhân đôi và chặn ở max, nhận %v", sleeps)
	}
}

func TestConsumerSkipsPoisonEventsAndCommits(t *testing.T) {
	reader := &fakeReader{records: []kafka.Message{
		{Offset: 1, Value: []byte("garbage")},
		event(2, "m2"),
	}}
	persister := &fakePersister{failures: 1, err: fmt.Errorf("%w: fk", model.ErrPermanent)}

	runConsumer(t, reader, persister)

	if fmt.Sprint(reader.committed) != "[1 2]" || len(persister.saved) != 0 {
		t.Fatalf("sự kiện hỏng phải bị bỏ qua và commit để không kẹt partition, nhận committed=%v saved=%v", reader.committed, persister.saved)
	}
}

func TestConsumerDoesNotCommitWhenStoppedMidRetry(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	reader := &fakeReader{records: []kafka.Message{event(3, "m1")}, cancel: cancel}
	persister := &fakePersister{failures: 100, err: errors.New("db down")}
	consumer := NewConsumer("chat", reader, NewChatMessageProcessor(persister), RetryPolicy{Initial: time.Millisecond, Max: time.Millisecond, PersistTimeout: time.Second})
	consumer.sleep = func(context.Context, time.Duration) error {
		cancel()
		return context.Canceled
	}

	if err := consumer.Run(ctx); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(reader.committed) != 0 {
		t.Fatal("chưa lưu được thì không được commit để consumer khác xử lý lại")
	}
}

type fakeTaskHandler struct {
	events []model.TaskStatusChanged
}

func (h *fakeTaskHandler) Handle(_ context.Context, event model.TaskStatusChanged) error {
	h.events = append(h.events, event)
	return nil
}

func TestTaskEventProcessorDecodesAndHandles(t *testing.T) {
	handler := &fakeTaskHandler{}
	value := []byte(`{"taskId":"t1","requestedBy":"alice","sourceMessageId":"m1","toStatus":{"name":"Done"},"changedAt":"2026-11-08T10:00:00Z"}`)

	if err := NewTaskEventProcessor(handler).Process(context.Background(), value); err != nil {
		t.Fatalf("process: %v", err)
	}
	if len(handler.events) != 1 || handler.events[0].TaskID != "t1" || handler.events[0].ToStatusName != "Done" {
		t.Fatalf("phải chuyển sự kiện đã decode, nhận %+v", handler.events)
	}
	if err := NewTaskEventProcessor(handler).Process(context.Background(), []byte("x")); !errors.Is(err, model.ErrPermanent) {
		t.Fatalf("payload hỏng là lỗi vĩnh viễn, nhận %v", err)
	}
}
