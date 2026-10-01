package service

import (
	"context"
	"sync"
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type stubMailService struct {
	mu       sync.Mutex
	calls    []string
	resultBy map[string]error
	accepted bool
	called   chan string
}

func (s *stubMailService) Subscribe(context.Context, model.SubscribeInput) (model.Subscription, error) {
	return model.Subscription{}, nil
}

func (s *stubMailService) Unsubscribe(context.Context, string) error {
	return nil
}

func (s *stubMailService) GetSubscription(context.Context, string) (model.Subscription, error) {
	return model.Subscription{}, nil
}

func (s *stubMailService) HandleNotification(_ context.Context, email string, _ string, _ string) (bool, error) {
	s.mu.Lock()
	s.calls = append(s.calls, email)
	err := s.resultBy[email]
	s.mu.Unlock()

	if s.called != nil {
		s.called <- email
	}
	if err != nil {
		return false, err
	}
	return s.accepted, nil
}

func (s *stubMailService) handled() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.calls...)
}

type stubNotificationSource struct {
	mu         sync.Mutex
	batches    [][]model.PulledNotice
	index      int
	recvErr    error
	ackErr     error
	ackCh      chan []string
	emptyCount int
	emptySeen  int
}

func (s *stubNotificationSource) Receive(ctx context.Context) ([]model.PulledNotice, error) {
	s.mu.Lock()
	if s.recvErr != nil {
		err := s.recvErr
		s.recvErr = nil
		s.mu.Unlock()
		return nil, err
	}
	if s.index < len(s.batches) {
		batch := s.batches[s.index]
		s.index++
		s.mu.Unlock()
		return batch, nil
	}
	if s.emptySeen < s.emptyCount {
		s.emptySeen++
		s.mu.Unlock()
		return nil, nil
	}
	s.mu.Unlock()

	<-ctx.Done()
	return nil, ctx.Err()
}

func (s *stubNotificationSource) Acknowledge(_ context.Context, ackIDs []string) error {
	if s.ackCh != nil {
		s.ackCh <- append([]string(nil), ackIDs...)
	}
	return s.ackErr
}

func runPullWorker(t *testing.T, worker Worker) (context.CancelFunc, chan struct{}) {
	t.Helper()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = worker.Run(ctx)
		close(done)
	}()
	t.Cleanup(func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("worker pull không dừng sau khi cancel")
		}
	})
	return cancel, done
}

func TestPullWorkerHandlesAndAcknowledges(t *testing.T) {
	source := &stubNotificationSource{
		batches: [][]model.PulledNotice{{
			{AckID: "ack-1", Notice: model.GmailNotice{EmailAddress: "user@example.com", HistoryID: "42", MessageID: "m1"}},
		}},
		ackCh: make(chan []string, 1),
	}
	mail := &stubMailService{accepted: true}

	runPullWorker(t, NewPullWorker(mail, source, testOptions()))

	select {
	case acked := <-source.ackCh:
		if len(acked) != 1 || acked[0] != "ack-1" {
			t.Fatalf("ack sai: %v", acked)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker không ack notice")
	}

	handled := mail.handled()
	if len(handled) != 1 || handled[0] != "user@example.com" {
		t.Fatalf("phải chuyển notice cho MailService, nhận %v", handled)
	}
}

func TestPullWorkerAcknowledgesUnknownMailbox(t *testing.T) {
	source := &stubNotificationSource{
		batches: [][]model.PulledNotice{{
			{AckID: "ack-1", Notice: model.GmailNotice{EmailAddress: "nobody@example.com", HistoryID: "42"}},
		}},
		ackCh: make(chan []string, 1),
	}
	mail := &stubMailService{accepted: false}

	runPullWorker(t, NewPullWorker(mail, source, testOptions()))

	select {
	case acked := <-source.ackCh:
		if len(acked) != 1 || acked[0] != "ack-1" {
			t.Fatalf("hộp thư lạ vẫn phải ack, nhận %v", acked)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker không ack hộp thư lạ")
	}
}

func TestPullWorkerAcknowledgesInvalidNotice(t *testing.T) {
	source := &stubNotificationSource{
		batches: [][]model.PulledNotice{{
			{AckID: "ack-bad"},
		}},
		ackCh: make(chan []string, 1),
	}
	mail := &stubMailService{accepted: true}

	runPullWorker(t, NewPullWorker(mail, source, testOptions()))

	select {
	case acked := <-source.ackCh:
		if len(acked) != 1 || acked[0] != "ack-bad" {
			t.Fatalf("notice hỏng phải bị ack để khỏi lặp vô hạn, nhận %v", acked)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker không ack notice hỏng")
	}

	if handled := mail.handled(); len(handled) != 0 {
		t.Fatalf("notice hỏng không được chuyển cho MailService, nhận %v", handled)
	}
}

func TestPullWorkerDoesNotAckWhenQueueFull(t *testing.T) {
	source := &stubNotificationSource{
		batches: [][]model.PulledNotice{{
			{AckID: "ack-1", Notice: model.GmailNotice{EmailAddress: "user@example.com", HistoryID: "42"}},
		}},
		ackCh: make(chan []string, 1),
	}
	mail := &stubMailService{
		resultBy: map[string]error{"user@example.com": model.ErrQueueFull},
		called:   make(chan string, 1),
	}

	runPullWorker(t, NewPullWorker(mail, source, testOptions()))

	select {
	case <-mail.called:
	case <-time.After(2 * time.Second):
		t.Fatal("worker không gọi MailService")
	}

	select {
	case acked := <-source.ackCh:
		t.Fatalf("queue đầy thì không được ack (để pubsub gửi lại), nhận %v", acked)
	case <-time.After(200 * time.Millisecond):
	}
}

func TestPullWorkerRetriesAfterReceiveError(t *testing.T) {
	source := &stubNotificationSource{
		recvErr: model.ErrNotConfigured,
		batches: [][]model.PulledNotice{{
			{AckID: "ack-1", Notice: model.GmailNotice{EmailAddress: "user@example.com", HistoryID: "42"}},
		}},
		ackCh: make(chan []string, 1),
	}
	options := testOptions()
	options.PullRetryDelay = 10 * time.Millisecond

	runPullWorker(t, NewPullWorker(&stubMailService{accepted: true}, source, options))

	select {
	case acked := <-source.ackCh:
		if len(acked) != 1 || acked[0] != "ack-1" {
			t.Fatalf("sau lỗi receive phải thử lại và xử lý, nhận %v", acked)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker không thử lại sau lỗi receive")
	}
}

type stepClock struct {
	mu   sync.Mutex
	now  time.Time
	step time.Duration
}

func (c *stepClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(c.step)
	return c.now
}

func TestPullWorkerHeartbeatWhenIdle(t *testing.T) {
	source := &stubNotificationSource{emptyCount: 30}
	worker := NewPullWorker(&stubMailService{}, source, testOptions()).(*pullWorker)
	worker.now = (&stepClock{now: time.Now(), step: time.Minute}).Now

	options := testOptions()
	options.HeartbeatInterval = 5 * time.Minute
	options.PullSubscription = "projects/demo/subscriptions/gmail-pull"
	worker.opts = options.withDefaults()

	beats := make(chan int, 4)
	worker.onHeartbeat = func(emptyPulls int, idle time.Duration) {
		beats <- emptyPulls
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	select {
	case got := <-beats:
		if got != 5 {
			t.Fatalf("heartbeat phải báo 5 lần pull rỗng, nhận %d", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("pull rỗng kéo dài mà không có heartbeat")
	}
}

func TestPullWorkerNoHeartbeatWhenDisabled(t *testing.T) {
	source := &stubNotificationSource{emptyCount: 30}
	worker := NewPullWorker(&stubMailService{}, source, testOptions()).(*pullWorker)
	worker.now = (&stepClock{now: time.Now(), step: time.Hour}).Now

	beats := make(chan int, 4)
	worker.onHeartbeat = func(emptyPulls int, idle time.Duration) {
		beats <- emptyPulls
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	select {
	case got := <-beats:
		t.Fatalf("HeartbeatInterval=0 thì không được heartbeat, nhận %d", got)
	case <-time.After(300 * time.Millisecond):
	}
}

func TestPullWorkerStopsOnContextCancel(t *testing.T) {
	source := &stubNotificationSource{}
	worker := NewPullWorker(&stubMailService{}, source, testOptions())

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = worker.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker pull không dừng khi cancel context")
	}
}
