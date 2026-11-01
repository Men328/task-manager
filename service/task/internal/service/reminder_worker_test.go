package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/task/internal/model"
)

type fakeReminderStore struct {
	queue    []model.Task
	released []string
}

func (s *fakeReminderStore) ClaimDueReminders(_ context.Context, limit int) ([]model.Task, error) {
	n := min(limit, len(s.queue))
	out := s.queue[:n]
	s.queue = s.queue[n:]
	return out, nil
}

func (s *fakeReminderStore) ReleaseReminder(_ context.Context, id string, _ time.Time) error {
	s.released = append(s.released, id)
	return nil
}

type fakeNoticeSender struct {
	sent []string
	fail map[string]bool
}

func (s *fakeNoticeSender) SendTaskReminder(_ context.Context, t model.Task) error {
	if s.fail[t.ID] {
		return errors.New("notification down")
	}
	s.sent = append(s.sent, t.ID)
	return nil
}

func claimed(id string, remindAt time.Time) model.Task {
	at := remindAt
	now := time.Now()
	return model.Task{ID: id, ProfileID: "u1", RemindAt: &at, RemindedAt: &now}
}

func TestReminderWorkerSendsSkipsLateAndReleasesFailures(t *testing.T) {
	now := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	store := &fakeReminderStore{queue: []model.Task{
		claimed("due", now.Add(-time.Minute)),
		claimed("too-late", now.Add(-7*time.Hour)),
		claimed("fails", now.Add(-time.Minute)),
	}}
	sender := &fakeNoticeSender{fail: map[string]bool{"fails": true}}
	worker := NewReminderWorker(store, sender, ReminderConfig{Batch: 10, MaxLate: 6 * time.Hour})
	worker.now = func() time.Time { return now }

	stats, n, err := worker.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if n != 3 || stats != (ReminderRunStats{Sent: 1, Skipped: 1, Failed: 1}) {
		t.Fatalf("stats = %+v (claimed %d)", stats, n)
	}
	if len(sender.sent) != 1 || sender.sent[0] != "due" {
		t.Fatalf("sent = %v, want [due]", sender.sent)
	}
	if len(store.released) != 1 || store.released[0] != "fails" {
		t.Fatalf("released = %v, want [fails] so it is retried", store.released)
	}
}

func TestReminderWorkerDrainsFullBatches(t *testing.T) {
	now := time.Now()
	store := &fakeReminderStore{}
	for _, id := range []string{"a", "b", "c", "d", "e"} {
		store.queue = append(store.queue, claimed(id, now))
	}
	sender := &fakeNoticeSender{}
	worker := NewReminderWorker(store, sender, ReminderConfig{Batch: 2, MaxLate: time.Hour})

	worker.drain(context.Background())
	if len(sender.sent) != 5 || len(store.queue) != 0 {
		t.Fatalf("drain sent %v, left %d", sender.sent, len(store.queue))
	}
}

func TestReminderWorkerStopsOnCancel(t *testing.T) {
	worker := NewReminderWorker(&fakeReminderStore{}, &fakeNoticeSender{}, ReminderConfig{Interval: time.Hour})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- worker.Run(ctx) }()
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not stop after cancel")
	}
}
