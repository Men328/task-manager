package service

import (
	"context"
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

func waitForHistoryID(t *testing.T, subs *stubSubscriptions, profileID string, want string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for {
		sub, err := subs.GetByProfileID(context.Background(), profileID)
		if err == nil && sub.HistoryID == want {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("checkpoint không đạt %q, nhận %q (err=%v)", want, sub.HistoryID, err)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func newTestWorker(
	subs *stubSubscriptions,
	gmail *stubGmail,
	analyzer MailAnalyzer,
	tasks TaskCreator,
	queue *Queue,
	opts Options,
) Worker {
	return NewWorker(
		subs,
		gmail,
		NewTokenManager(subs, &stubRefresher{}),
		analyzer,
		tasks,
		&stubWorkspaces{id: "ws-1"},
		queue,
		opts,
	)
}

func TestWorkerCreatesTaskFromNotification(t *testing.T) {
	subs := newStubSubscriptions(model.Subscription{
		ProfileID:            "p1",
		Email:                "user@example.com",
		AccessToken:          "access",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		HistoryID:            "100",
	})
	gmail := &stubGmail{
		messageIDs:      []string{"m1"},
		latestHistoryID: "120",
		messages: map[string]model.EmailMessage{
			"m1": {ID: "m1", Subject: "Nộp báo cáo", Body: "Hạn thứ sáu"},
		},
	}
	created := make(chan model.TaskInput, 1)
	queue := NewQueue(4)
	worker := newTestWorker(subs, gmail, &stubAnalyzer{draft: model.TaskDraft{
		Actionable:  true,
		Title:       "Nộp báo cáo",
		Description: "Gửi báo cáo trước thứ sáu",
		Priority:    model.PriorityHigh,
	}}, &stubTasks{created: created}, queue, testOptions())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	queue.Enqueue(model.Notification{ProfileID: "p1", Email: "user@example.com", HistoryID: "110", MessageID: "m1"})

	select {
	case input := <-created:
		if input.ProfileID != "p1" {
			t.Fatalf("task phải thuộc đúng profile, nhận %q", input.ProfileID)
		}
		if input.WorkspaceID != "ws-1" {
			t.Fatalf("task phải dùng workspace mặc định, nhận %q", input.WorkspaceID)
		}
		if input.Title != "Nộp báo cáo" || input.Priority != model.PriorityHigh {
			t.Fatalf("task map sai draft: %+v", input)
		}
		if input.Source != "m1" {
			t.Fatalf("task phải ghi nguồn message, nhận %q", input.Source)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker không tạo task")
	}

	waitForHistoryID(t, subs, "p1", "120")
}

func TestWorkerSkipsNonActionableEmail(t *testing.T) {
	subs := newStubSubscriptions(model.Subscription{
		ProfileID:            "p1",
		Email:                "user@example.com",
		AccessToken:          "access",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		HistoryID:            "100",
	})
	gmail := &stubGmail{
		messageIDs:      []string{"m1"},
		latestHistoryID: "130",
		messages: map[string]model.EmailMessage{
			"m1": {ID: "m1", Subject: "Newsletter", Body: "Khuyến mãi"},
		},
	}
	created := make(chan model.TaskInput, 1)
	queue := NewQueue(4)
	worker := newTestWorker(subs, gmail, &stubAnalyzer{draft: model.TaskDraft{Actionable: false}}, &stubTasks{created: created}, queue, testOptions())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	queue.Enqueue(model.Notification{ProfileID: "p1", HistoryID: "110", MessageID: "m1"})

	waitForHistoryID(t, subs, "p1", "130")

	select {
	case input := <-created:
		t.Fatalf("email không cần hành động không được tạo task: %+v", input)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestWorkerSkipsAnalysisWhenAnalyzerDisabled(t *testing.T) {
	subs := newStubSubscriptions(model.Subscription{
		ProfileID:            "p1",
		AccessToken:          "access",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		HistoryID:            "100",
	})
	gmail := &stubGmail{messageIDs: []string{"m1"}, latestHistoryID: "140"}
	created := make(chan model.TaskInput, 1)
	queue := NewQueue(4)
	options := testOptions()
	options.AnalyzerEnabled = false
	worker := newTestWorker(subs, gmail, &stubAnalyzer{draft: model.TaskDraft{Actionable: true, Title: "X"}}, &stubTasks{created: created}, queue, options)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	queue.Enqueue(model.Notification{ProfileID: "p1", HistoryID: "110", MessageID: "m1"})

	waitForHistoryID(t, subs, "p1", "140")

	select {
	case input := <-created:
		t.Fatalf("tắt analyzer thì không được tạo task: %+v", input)
	default:
	}
}

func TestWorkerResetsCheckpointWhenHistoryGone(t *testing.T) {
	subs := newStubSubscriptions(model.Subscription{
		ProfileID:            "p1",
		Email:                "user@example.com",
		AccessToken:          "access",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		HistoryID:            "100",
	})
	gmail := &stubGmail{historyErr: model.ErrHistoryGone, profileHistoryID: "999"}
	created := make(chan model.TaskInput, 1)
	queue := NewQueue(4)
	worker := newTestWorker(subs, gmail, &stubAnalyzer{draft: model.TaskDraft{Actionable: true, Title: "X"}}, &stubTasks{created: created}, queue, testOptions())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	queue.Enqueue(model.Notification{ProfileID: "p1", HistoryID: "110", MessageID: "m1"})

	waitForHistoryID(t, subs, "p1", "999")

	select {
	case input := <-created:
		t.Fatalf("checkpoint hỏng thì không được tạo task từ khoảng đã mất: %+v", input)
	case <-time.After(100 * time.Millisecond):
	}
}
