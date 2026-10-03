package service

import (
	"context"
	"strings"
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
	return newTestWorkerWith(subs, gmail, analyzer, tasks, &stubSchedules{}, &stubEvents{}, &stubBacklogs{}, newStubStorage(), queue, opts)
}

func newTestWorkerWith(
	subs *stubSubscriptions,
	gmail *stubGmail,
	analyzer MailAnalyzer,
	tasks TaskCreator,
	schedules ScheduleCreator,
	events EventCreator,
	backlogs BacklogCreator,
	storage BlobStore,
	queue *Queue,
	opts Options,
) Worker {
	return NewWorker(
		subs,
		gmail,
		NewTokenManager(subs, &stubRefresher{}),
		analyzer,
		tasks,
		schedules,
		events,
		backlogs,
		storage,
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
	worker := newTestWorker(subs, gmail, &stubAnalyzer{draft: model.MailDraft{
		Category:    model.CategoryTask,
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
	worker := newTestWorker(subs, gmail, &stubAnalyzer{draft: model.MailDraft{Category: model.CategoryTask, Actionable: false}}, &stubTasks{created: created}, queue, testOptions())

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
	worker := newTestWorker(subs, gmail, &stubAnalyzer{draft: model.MailDraft{Category: model.CategoryTask, Actionable: true, Title: "X"}}, &stubTasks{created: created}, queue, options)

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
	worker := newTestWorker(subs, gmail, &stubAnalyzer{draft: model.MailDraft{Category: model.CategoryTask, Actionable: true, Title: "X"}}, &stubTasks{created: created}, queue, testOptions())

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

func ruleOptions() Options {
	options := testOptions()
	options.AnalyzerEnabled = false
	options.RuleFallbackEnabled = true
	return options
}

func newSubscriptionFor(profileID string) *stubSubscriptions {
	return newStubSubscriptions(model.Subscription{
		ProfileID:            profileID,
		Email:                "user@example.com",
		AccessToken:          "access",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
		HistoryID:            "100",
	})
}

func TestWorkerDispatchesScheduleByRule(t *testing.T) {
	subs := newSubscriptionFor("p1")
	gmail := &stubGmail{
		messageIDs:      []string{"m1"},
		latestHistoryID: "160",
		messages: map[string]model.EmailMessage{
			"m1": {ID: "m1", Subject: "Họp nhóm lúc 2026-10-05 09:00", Body: "Mời họp nhóm dự án"},
		},
	}
	schedules := &stubSchedules{created: make(chan model.ScheduleInput, 1)}
	queue := NewQueue(4)
	worker := newTestWorkerWith(subs, gmail, &stubAnalyzer{}, &stubTasks{}, schedules, &stubEvents{}, &stubBacklogs{}, newStubStorage(), queue, ruleOptions())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	queue.Enqueue(model.Notification{ProfileID: "p1", HistoryID: "110", MessageID: "m1"})

	select {
	case input := <-schedules.created:
		if input.ProfileID != "p1" {
			t.Fatalf("lịch phải thuộc đúng profile, nhận %q", input.ProfileID)
		}
		if input.StartAt.Year() != 2026 || input.StartAt.Month() != time.October || input.StartAt.Day() != 5 {
			t.Fatalf("rule phải bóc được thời gian bắt đầu, nhận %v", input.StartAt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker không tạo lịch từ email")
	}
}

func TestWorkerDispatchesEventByRule(t *testing.T) {
	subs := newSubscriptionFor("p1")
	gmail := &stubGmail{
		messageIDs:      []string{"m1"},
		latestHistoryID: "170",
		messages: map[string]model.EmailMessage{
			"m1": {ID: "m1", Subject: "Thư mời tham dự hội thảo", Body: "Sự kiện ngày 12/11/2026 14:00 tại hội trường"},
		},
	}
	events := &stubEvents{created: make(chan model.EventInput, 1)}
	queue := NewQueue(4)
	worker := newTestWorkerWith(subs, gmail, &stubAnalyzer{}, &stubTasks{}, &stubSchedules{}, events, &stubBacklogs{}, newStubStorage(), queue, ruleOptions())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	queue.Enqueue(model.Notification{ProfileID: "p1", HistoryID: "110", MessageID: "m1"})

	select {
	case input := <-events.created:
		if input.Source != "m1" {
			t.Fatalf("sự kiện phải ghi nguồn message, nhận %q", input.Source)
		}
		if input.StartAt.Month() != time.November || input.StartAt.Day() != 12 {
			t.Fatalf("rule phải bóc được thời gian sự kiện, nhận %v", input.StartAt)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker không tạo sự kiện từ email")
	}
}

func TestWorkerSendsUnclassifiedEmailToBacklog(t *testing.T) {
	subs := newSubscriptionFor("p1")
	gmail := &stubGmail{
		messageIDs:      []string{"m1"},
		latestHistoryID: "180",
		messages: map[string]model.EmailMessage{
			"m1": {ID: "m1", Subject: "Newsletter tháng 10", Body: "Khuyến mãi cuối năm"},
		},
	}
	backlogs := &stubBacklogs{created: make(chan model.BacklogInput, 1)}
	queue := NewQueue(4)
	worker := newTestWorkerWith(subs, gmail, &stubAnalyzer{}, &stubTasks{}, &stubSchedules{}, &stubEvents{}, backlogs, newStubStorage(), queue, ruleOptions())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	queue.Enqueue(model.Notification{ProfileID: "p1", HistoryID: "110", MessageID: "m1"})

	select {
	case input := <-backlogs.created:
		if input.Category != string(model.CategoryOther) {
			t.Fatalf("email không khớp rule phải vào backlog với category other, nhận %q", input.Category)
		}
		if input.Reason != "no_rule_matched" {
			t.Fatalf("reason phải là no_rule_matched, nhận %q", input.Reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker không đẩy email vào backlog")
	}
}

func TestWorkerRoutesMissingScheduleTimeToBacklog(t *testing.T) {
	subs := newSubscriptionFor("p1")
	gmail := &stubGmail{
		messageIDs:      []string{"m1"},
		latestHistoryID: "190",
		messages: map[string]model.EmailMessage{
			"m1": {ID: "m1", Subject: "Lịch họp", Body: "Chưa chốt giờ"},
		},
	}
	backlogs := &stubBacklogs{created: make(chan model.BacklogInput, 1)}
	queue := NewQueue(4)
	analyzer := &stubAnalyzer{draft: model.MailDraft{Category: model.CategorySchedule, Actionable: true, Title: "Lịch họp"}}
	worker := newTestWorkerWith(subs, gmail, analyzer, &stubTasks{}, &stubSchedules{}, &stubEvents{}, backlogs, newStubStorage(), queue, testOptions())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	queue.Enqueue(model.Notification{ProfileID: "p1", HistoryID: "110", MessageID: "m1"})

	select {
	case input := <-backlogs.created:
		if input.Reason != "missing_schedule_time" {
			t.Fatalf("thiếu thời gian phải vào backlog, nhận reason %q", input.Reason)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("worker không đẩy email thiếu dữ kiện vào backlog")
	}
}

func TestWorkerArchivesEmailAndAttachment(t *testing.T) {
	subs := newSubscriptionFor("p1")
	gmail := &stubGmail{
		messageIDs:      []string{"m1"},
		latestHistoryID: "200",
		messages: map[string]model.EmailMessage{
			"m1": {
				ID:      "m1",
				Subject: "Newsletter tháng 10",
				Body:    "Khuyến mãi cuối năm",
				Attachments: []model.EmailAttachment{
					{Filename: "bang-gia.pdf", MimeType: "application/pdf", AttachmentID: "att1", Size: 9},
				},
			},
		},
		attachments: map[string][]byte{"att1": []byte("pdf-bytes")},
	}
	storage := newStubStorage()
	backlogs := &stubBacklogs{created: make(chan model.BacklogInput, 1)}
	queue := NewQueue(4)
	options := ruleOptions()
	options.ArchiveEnabled = true
	worker := newTestWorkerWith(subs, gmail, &stubAnalyzer{}, &stubTasks{}, &stubSchedules{}, &stubEvents{}, backlogs, storage, queue, options)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = worker.Run(ctx) }()

	queue.Enqueue(model.Notification{ProfileID: "p1", HistoryID: "110", MessageID: "m1"})

	var objectKey string
	select {
	case input := <-backlogs.created:
		objectKey = input.ObjectKey
	case <-time.After(2 * time.Second):
		t.Fatal("worker không đẩy email vào backlog")
	}
	if objectKey == "" {
		t.Fatal("backlog phải giữ object key của email gốc trên storage")
	}

	objects := storage.stored()
	if len(objects) != 2 {
		t.Fatalf("phải lưu email gốc + attachment, nhận %d object", len(objects))
	}
	foundEmail := false
	foundAttachment := false
	for key, data := range objects {
		if strings.Contains(key, "m1.json") && strings.Contains(string(data), "Newsletter") {
			foundEmail = true
		}
		if strings.Contains(key, "bang-gia.pdf") && string(data) == "pdf-bytes" {
			foundAttachment = true
		}
	}
	if !foundEmail || !foundAttachment {
		t.Fatalf("thiếu object: email=%v attachment=%v", foundEmail, foundAttachment)
	}
}
