package repository

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	backlogv1 "taskmanager/common/gen/go/backlog/v1"
	calendarv1 "taskmanager/common/gen/go/calendar/v1"
	eventv1 "taskmanager/common/gen/go/event/v1"
	taskv1 "taskmanager/common/gen/go/task/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type captureTaskServer struct {
	taskv1.UnimplementedTaskServiceServer
	request *taskv1.CreateTaskRequest
	err     error
}

func (s *captureTaskServer) CreateTask(_ context.Context, req *taskv1.CreateTaskRequest) (*taskv1.CreateTaskResponse, error) {
	s.request = req
	if s.err != nil {
		return nil, s.err
	}
	return &taskv1.CreateTaskResponse{Task: &taskv1.Task{Id: "task-1", Title: req.GetTitle()}}, nil
}

type captureCalendarServer struct {
	calendarv1.UnimplementedCalendarServiceServer
	request *calendarv1.CreateScheduleRequest
	err     error
}

func (s *captureCalendarServer) CreateSchedule(_ context.Context, req *calendarv1.CreateScheduleRequest) (*calendarv1.CreateScheduleResponse, error) {
	s.request = req
	if s.err != nil {
		return nil, s.err
	}
	return &calendarv1.CreateScheduleResponse{Schedule: &calendarv1.Schedule{Id: "schedule-1", Title: req.GetTitle()}}, nil
}

type captureEventServer struct {
	eventv1.UnimplementedEventServiceServer
	request *eventv1.CreateEventRequest
	err     error
}

func (s *captureEventServer) CreateEvent(_ context.Context, req *eventv1.CreateEventRequest) (*eventv1.CreateEventResponse, error) {
	s.request = req
	if s.err != nil {
		return nil, s.err
	}
	return &eventv1.CreateEventResponse{Event: &eventv1.Event{Id: "event-1", Title: req.GetTitle()}}, nil
}

type captureBacklogServer struct {
	backlogv1.UnimplementedBacklogServiceServer
	request *backlogv1.CreateBacklogRequest
	err     error
}

func (s *captureBacklogServer) CreateBacklog(_ context.Context, req *backlogv1.CreateBacklogRequest) (*backlogv1.CreateBacklogResponse, error) {
	s.request = req
	if s.err != nil {
		return nil, s.err
	}
	return &backlogv1.CreateBacklogResponse{Backlog: &backlogv1.Backlog{Id: "backlog-1", Title: req.GetTitle()}}, nil
}

func startBufconnServer(t *testing.T, register func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	register(server)
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(server.Stop)

	conn, err := grpc.NewClient(
		"passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return listener.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("dial bufconn: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestTaskClientSendsDraftAndReturnsRef(t *testing.T) {
	server := &captureTaskServer{}
	conn := startBufconnServer(t, func(s *grpc.Server) { taskv1.RegisterTaskServiceServer(s, server) })
	client := NewTaskClient(taskv1.NewTaskServiceClient(conn), time.Second)
	due := time.Date(2026, 10, 9, 17, 0, 0, 0, time.UTC)

	ref, err := client.Create(context.Background(), model.TaskInput{
		ProfileID:   "p1",
		Title:       "Nộp báo cáo",
		Description: "Gửi trước thứ sáu",
		Priority:    model.PriorityHigh,
		DueAt:       &due,
		Source:      "m1",
	})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	if ref.ID != "task-1" || ref.Title != "Nộp báo cáo" {
		t.Fatalf("ref sai: %+v", ref)
	}
	if server.request.GetProfileId() != "p1" || server.request.GetTitle() != "Nộp báo cáo" {
		t.Fatalf("request sai: %+v", server.request)
	}
	if server.request.GetPriority() != taskv1.TaskPriority(model.PriorityHigh) {
		t.Fatalf("priority sai: %v", server.request.GetPriority())
	}
	if server.request.GetDueAt().AsTime().UTC() != due {
		t.Fatalf("due_at sai: %v", server.request.GetDueAt())
	}
}

func TestTaskClientWrapsServerError(t *testing.T) {
	server := &captureTaskServer{err: errors.New("boom")}
	conn := startBufconnServer(t, func(s *grpc.Server) { taskv1.RegisterTaskServiceServer(s, server) })
	client := NewTaskClient(taskv1.NewTaskServiceClient(conn), time.Second)

	_, err := client.Create(context.Background(), model.TaskInput{ProfileID: "p1", Title: "x"})
	if !errors.Is(err, model.ErrTaskCreate) {
		t.Fatalf("phải bọc ErrTaskCreate, nhận %v", err)
	}
}

func TestScheduleClientSendsSchedule(t *testing.T) {
	server := &captureCalendarServer{}
	conn := startBufconnServer(t, func(s *grpc.Server) { calendarv1.RegisterCalendarServiceServer(s, server) })
	client := NewScheduleClient(calendarv1.NewCalendarServiceClient(conn), time.Second)
	start := time.Date(2026, 10, 5, 1, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)

	ref, err := client.Create(context.Background(), model.ScheduleInput{
		ProfileID:   "p1",
		Title:       "Họp nhóm",
		Description: "Standup",
		Location:    "Zoom",
		StartAt:     start,
		EndAt:       &end,
		AllDay:      false,
	})
	if err != nil {
		t.Fatalf("create schedule: %v", err)
	}
	if ref.ID != "schedule-1" {
		t.Fatalf("ref sai: %+v", ref)
	}
	if server.request.GetProfileId() != "p1" || server.request.GetLocation() != "Zoom" {
		t.Fatalf("request sai: %+v", server.request)
	}
	if server.request.GetStartAt().AsTime().UTC() != start || server.request.GetEndAt().AsTime().UTC() != end {
		t.Fatalf("thời gian sai: %v - %v", server.request.GetStartAt(), server.request.GetEndAt())
	}
}

func TestEventClientSendsPlannedEventWithSource(t *testing.T) {
	server := &captureEventServer{}
	conn := startBufconnServer(t, func(s *grpc.Server) { eventv1.RegisterEventServiceServer(s, server) })
	client := NewEventClient(eventv1.NewEventServiceClient(conn), time.Second)
	start := time.Date(2026, 11, 12, 7, 0, 0, 0, time.UTC)

	ref, err := client.Create(context.Background(), model.EventInput{
		ProfileID: "p1",
		Title:     "Hội thảo AI",
		Location:  "Hà Nội",
		StartAt:   start,
		Source:    "m1",
	})
	if err != nil {
		t.Fatalf("create event: %v", err)
	}
	if ref.ID != "event-1" {
		t.Fatalf("ref sai: %+v", ref)
	}
	if server.request.GetStatus() != eventv1.EventStatus_EVENT_STATUS_PLANNED {
		t.Fatalf("status phải PLANNED, nhận %v", server.request.GetStatus())
	}
	if server.request.GetSource() != "m1" {
		t.Fatalf("source phải là message id, nhận %q", server.request.GetSource())
	}
	if server.request.GetStartAt().AsTime().UTC() != start {
		t.Fatalf("start_at sai: %v", server.request.GetStartAt())
	}
}

func TestBacklogClientSendsClassification(t *testing.T) {
	server := &captureBacklogServer{}
	conn := startBufconnServer(t, func(s *grpc.Server) { backlogv1.RegisterBacklogServiceServer(s, server) })
	client := NewBacklogClient(backlogv1.NewBacklogServiceClient(conn), time.Second)

	ref, err := client.Create(context.Background(), model.BacklogInput{
		ProfileID:   "p1",
		Title:       "Newsletter",
		Description: "Khuyến mãi",
		Sender:      "promo@shop.vn",
		Source:      "m1",
		Category:    string(model.CategoryOther),
		Reason:      "no_rule_matched",
		ObjectKey:   "mail/p1/20261003/m1.json",
	})
	if err != nil {
		t.Fatalf("create backlog: %v", err)
	}
	if ref.ID != "backlog-1" {
		t.Fatalf("ref sai: %+v", ref)
	}
	if server.request.GetCategory() != "other" || server.request.GetReason() != "no_rule_matched" {
		t.Fatalf("phân loại sai: %+v", server.request)
	}
	if server.request.GetObjectKey() != "mail/p1/20261003/m1.json" || server.request.GetSender() != "promo@shop.vn" {
		t.Fatalf("metadata sai: %+v", server.request)
	}
}

func TestBacklogClientWrapsServerError(t *testing.T) {
	server := &captureBacklogServer{err: errors.New("boom")}
	conn := startBufconnServer(t, func(s *grpc.Server) { backlogv1.RegisterBacklogServiceServer(s, server) })
	client := NewBacklogClient(backlogv1.NewBacklogServiceClient(conn), time.Second)

	_, err := client.Create(context.Background(), model.BacklogInput{ProfileID: "p1", Title: "x"})
	if !errors.Is(err, model.ErrBacklogCreate) {
		t.Fatalf("phải bọc ErrBacklogCreate, nhận %v", err)
	}
}
