package repository

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	backlogv1 "taskmanager/common/gen/go/backlog/v1"
	calendarv1 "taskmanager/common/gen/go/calendar/v1"
	eventv1 "taskmanager/common/gen/go/event/v1"
	taskv1 "taskmanager/common/gen/go/task/v1"
)

type fakeTasks struct {
	taskv1.TaskServiceClient
	req *taskv1.ListTasksRequest
}

func (f *fakeTasks) ListTasks(_ context.Context, in *taskv1.ListTasksRequest, _ ...grpc.CallOption) (*taskv1.ListTasksResponse, error) {
	f.req = in
	return &taskv1.ListTasksResponse{}, nil
}

type fakeEvents struct {
	eventv1.EventServiceClient
	req *eventv1.ListEventsRequest
}

func (f *fakeEvents) ListEvents(_ context.Context, in *eventv1.ListEventsRequest, _ ...grpc.CallOption) (*eventv1.ListEventsResponse, error) {
	f.req = in
	return &eventv1.ListEventsResponse{}, nil
}

type fakeSchedules struct {
	calendarv1.CalendarServiceClient
	req *calendarv1.ListSchedulesRequest
}

func (f *fakeSchedules) ListSchedules(_ context.Context, in *calendarv1.ListSchedulesRequest, _ ...grpc.CallOption) (*calendarv1.ListSchedulesResponse, error) {
	f.req = in
	return &calendarv1.ListSchedulesResponse{}, nil
}

type fakeBacklogs struct {
	backlogv1.BacklogServiceClient
	req *backlogv1.ListBacklogsRequest
}

func (f *fakeBacklogs) ListBacklogs(_ context.Context, in *backlogv1.ListBacklogsRequest, _ ...grpc.CallOption) (*backlogv1.ListBacklogsResponse, error) {
	f.req = in
	return &backlogv1.ListBacklogsResponse{}, nil
}

func TestFetchTasksSendsProject(t *testing.T) {
	fake := &fakeTasks{}
	source := NewTaskDataSource(fake, nil, 0)
	if _, err := source.FetchTasks(context.Background(), "prof", "p1", true); err != nil {
		t.Fatal(err)
	}
	if fake.req.GetProjectId() != "p1" || fake.req.GetProfileId() != "prof" || !fake.req.GetIncludeArchived() {
		t.Fatalf("unexpected request %v", fake.req)
	}
}

func TestActivitySourceSendsProjectExceptBacklog(t *testing.T) {
	events, schedules, backlogs := &fakeEvents{}, &fakeSchedules{}, &fakeBacklogs{}
	source := NewActivityDataSource(events, schedules, backlogs, 0)
	ctx := context.Background()

	if _, err := source.FetchEvents(ctx, "prof", "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.FetchSchedules(ctx, "prof", "p1"); err != nil {
		t.Fatal(err)
	}
	if _, err := source.FetchBacklogs(ctx, "prof"); err != nil {
		t.Fatal(err)
	}
	if events.req.GetProjectId() != "p1" {
		t.Fatalf("events project = %q", events.req.GetProjectId())
	}
	if schedules.req.GetProjectId() != "p1" {
		t.Fatalf("schedules project = %q", schedules.req.GetProjectId())
	}
	if backlogs.req.GetProfileId() != "prof" {
		t.Fatalf("backlog request %v", backlogs.req)
	}
}
