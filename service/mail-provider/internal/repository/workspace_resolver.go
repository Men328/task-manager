package repository

import (
	"context"
	"fmt"
	"time"

	workspacev1 "taskmanager/common/gen/go/workspace/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type workspaceResolver struct {
	client  workspacev1.WorkspaceServiceClient
	timeout time.Duration
}

func NewWorkspaceResolver(client workspacev1.WorkspaceServiceClient, timeout time.Duration) *workspaceResolver {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &workspaceResolver{client: client, timeout: timeout}
}

func (r *workspaceResolver) DefaultWorkspaceID(ctx context.Context, profileID string) (string, error) {
	callCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	response, err := r.client.ListWorkspaces(callCtx, &workspacev1.ListWorkspacesRequest{OwnerProfileId: profileID})
	if err != nil {
		return "", fmt.Errorf("%w: %v", model.ErrNoWorkspace, err)
	}

	for _, workspace := range response.GetWorkspaces() {
		if workspace.GetId() != "" {
			return workspace.GetId(), nil
		}
	}
	return "", model.ErrNoWorkspace
}
