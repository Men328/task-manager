package repository

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	projectv1 "taskmanager/common/gen/go/project/v1"
	"taskmanager/service/calendar/internal/model"
)

type projectChecker struct {
	client  projectv1.ProjectServiceClient
	timeout time.Duration
}

func NewProjectChecker(client projectv1.ProjectServiceClient, timeout time.Duration) *projectChecker {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &projectChecker{client: client, timeout: timeout}
}

// CheckAssignable xác minh project tồn tại, thuộc profile và chưa archive.
func (c *projectChecker) CheckAssignable(ctx context.Context, profileID, projectID string) error {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	resp, err := c.client.GetProject(callCtx, &projectv1.GetProjectRequest{Id: projectID})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return model.NewError(model.ErrorKindProjectNotFound, "project %s not found", projectID)
		}
		return fmt.Errorf("kiểm tra project %s: %w", projectID, err)
	}

	p := resp.GetProject()
	if p == nil || p.GetProfileId() != profileID {
		return model.NewError(model.ErrorKindProjectNotFound, "project %s not found", projectID)
	}
	if p.GetIsArchived() {
		return model.NewError(model.ErrorKindProjectArchived, "project %s is archived", projectID)
	}
	return nil
}
