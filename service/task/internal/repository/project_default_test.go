package repository

import (
	"context"
	"testing"
	"time"

	"google.golang.org/grpc"

	projectv1 "taskmanager/common/gen/go/project/v1"
)

type fakeDefaultProjectClient struct {
	projectv1.ProjectServiceClient
	projects []*projectv1.Project
	seeded   int
}

func (f *fakeDefaultProjectClient) ListProjects(context.Context, *projectv1.ListProjectsRequest, ...grpc.CallOption) (*projectv1.ListProjectsResponse, error) {
	return &projectv1.ListProjectsResponse{Projects: f.projects}, nil
}

func (f *fakeDefaultProjectClient) SeedDefaultProject(_ context.Context, in *projectv1.SeedDefaultProjectRequest, _ ...grpc.CallOption) (*projectv1.SeedDefaultProjectResponse, error) {
	f.seeded++
	return &projectv1.SeedDefaultProjectResponse{Project: &projectv1.Project{Id: "seeded-" + in.GetProfileId()}}, nil
}

func TestDefaultProjectIDPicksDefault(t *testing.T) {
	client := &fakeDefaultProjectClient{projects: []*projectv1.Project{{Id: "a"}, {Id: "b", IsDefault: true}}}
	got, err := NewProjectChecker(client, time.Second).DefaultProjectID(context.Background(), "p")
	if err != nil || got != "b" || client.seeded != 0 {
		t.Fatalf("phải chọn project mặc định, nhận %q %v (seed %d)", got, err, client.seeded)
	}
}

func TestDefaultProjectIDSeedsWhenMissing(t *testing.T) {
	client := &fakeDefaultProjectClient{projects: []*projectv1.Project{{Id: "a"}}}
	got, err := NewProjectChecker(client, time.Second).DefaultProjectID(context.Background(), "p")
	if err != nil || got != "seeded-p" || client.seeded != 1 {
		t.Fatalf("thiếu mặc định thì phải seed, nhận %q %v", got, err)
	}
}
