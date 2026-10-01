package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"

	workspacev1 "taskmanager/common/gen/go/workspace/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type stubWorkspaceClient struct {
	workspaces []*workspacev1.Workspace
	err        error
	requests   []*workspacev1.ListWorkspacesRequest
}

func (c *stubWorkspaceClient) CreateWorkspace(_ context.Context, _ *workspacev1.CreateWorkspaceRequest, _ ...grpc.CallOption) (*workspacev1.CreateWorkspaceResponse, error) {
	return &workspacev1.CreateWorkspaceResponse{}, nil
}

func (c *stubWorkspaceClient) GetWorkspace(_ context.Context, _ *workspacev1.GetWorkspaceRequest, _ ...grpc.CallOption) (*workspacev1.GetWorkspaceResponse, error) {
	return &workspacev1.GetWorkspaceResponse{}, nil
}

func (c *stubWorkspaceClient) ListWorkspaces(_ context.Context, in *workspacev1.ListWorkspacesRequest, _ ...grpc.CallOption) (*workspacev1.ListWorkspacesResponse, error) {
	c.requests = append(c.requests, in)
	if c.err != nil {
		return nil, c.err
	}
	return &workspacev1.ListWorkspacesResponse{Workspaces: c.workspaces}, nil
}

func (c *stubWorkspaceClient) UpdateWorkspace(_ context.Context, _ *workspacev1.UpdateWorkspaceRequest, _ ...grpc.CallOption) (*workspacev1.UpdateWorkspaceResponse, error) {
	return &workspacev1.UpdateWorkspaceResponse{}, nil
}

func TestWorkspaceResolverReturnsFirstWorkspace(t *testing.T) {
	client := &stubWorkspaceClient{workspaces: []*workspacev1.Workspace{
		{Id: "ws-first", IsDefault: false},
		{Id: "ws-default", IsDefault: true},
		{Id: "ws-third"},
	}}
	resolver := NewWorkspaceResolver(client, time.Second)

	id, err := resolver.DefaultWorkspaceID(context.Background(), "p1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if id != "ws-first" {
		t.Fatalf("phải lấy workspace đầu tiên, nhận %q", id)
	}
	if len(client.requests) != 1 {
		t.Fatalf("phải gọi ListWorkspaces đúng 1 lần, nhận %d", len(client.requests))
	}
	if client.requests[0].GetOwnerProfileId() != "p1" {
		t.Fatalf("phải lọc theo owner, nhận %q", client.requests[0].GetOwnerProfileId())
	}
	if client.requests[0].GetIncludeArchived() {
		t.Fatal("không được yêu cầu cả workspace đã archive")
	}
}

func TestWorkspaceResolverSkipsEmptyID(t *testing.T) {
	client := &stubWorkspaceClient{workspaces: []*workspacev1.Workspace{
		{Id: ""},
		{Id: "ws-second"},
	}}
	resolver := NewWorkspaceResolver(client, time.Second)

	id, err := resolver.DefaultWorkspaceID(context.Background(), "p1")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if id != "ws-second" {
		t.Fatalf("phải bỏ qua workspace không có id, nhận %q", id)
	}
}

func TestWorkspaceResolverFailsWhenProfileHasNoWorkspace(t *testing.T) {
	resolver := NewWorkspaceResolver(&stubWorkspaceClient{}, time.Second)

	if _, err := resolver.DefaultWorkspaceID(context.Background(), "p1"); !errors.Is(err, model.ErrNoWorkspace) {
		t.Fatalf("profile không có workspace phải là ErrNoWorkspace, nhận %v", err)
	}
}

func TestWorkspaceResolverWrapsTransportError(t *testing.T) {
	client := &stubWorkspaceClient{err: errors.New("workspace service offline")}
	resolver := NewWorkspaceResolver(client, time.Second)

	if _, err := resolver.DefaultWorkspaceID(context.Background(), "p1"); !errors.Is(err, model.ErrNoWorkspace) {
		t.Fatalf("lỗi gọi workspace service phải là ErrNoWorkspace, nhận %v", err)
	}
}
