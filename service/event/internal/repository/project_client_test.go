package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	projectv1 "taskmanager/common/gen/go/project/v1"
	"taskmanager/service/event/internal/model"
)

type fakeProjectClient struct {
	projectv1.ProjectServiceClient
	resp *projectv1.GetProjectResponse
	err  error
}

func (f *fakeProjectClient) GetProject(context.Context, *projectv1.GetProjectRequest, ...grpc.CallOption) (*projectv1.GetProjectResponse, error) {
	return f.resp, f.err
}

func TestProjectCheckerCheckAssignable(t *testing.T) {
	other := errors.New("boom")
	cases := []struct {
		name     string
		client   *fakeProjectClient
		wantKind model.ErrorKind
		wantErr  error
		wantOK   bool
	}{
		{"ok", &fakeProjectClient{resp: &projectv1.GetProjectResponse{Project: &projectv1.Project{Id: "j", ProfileId: "p"}}}, "", nil, true},
		{"not found", &fakeProjectClient{err: status.Error(codes.NotFound, "x")}, model.ErrorKindProjectNotFound, nil, false},
		{"other profile", &fakeProjectClient{resp: &projectv1.GetProjectResponse{Project: &projectv1.Project{Id: "j", ProfileId: "q"}}}, model.ErrorKindProjectNotFound, nil, false},
		{"archived", &fakeProjectClient{resp: &projectv1.GetProjectResponse{Project: &projectv1.Project{Id: "j", ProfileId: "p", IsArchived: true}}}, model.ErrorKindProjectArchived, nil, false},
		{"other error", &fakeProjectClient{err: other}, "", other, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := NewProjectChecker(tc.client, time.Second).CheckAssignable(context.Background(), "p", "j")
			if tc.wantOK {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("want wrapped %v, got %v", tc.wantErr, err)
				}
				return
			}
			var de *model.Error
			if !errors.As(err, &de) || de.Kind != tc.wantKind {
				t.Fatalf("want kind %s, got %v", tc.wantKind, err)
			}
		})
	}
}
