package repository

import (
	"context"
	"fmt"
	"time"

	projectv1 "taskmanager/common/gen/go/project/v1"
	"taskmanager/service/mail-provider/internal/model"
)

type projectClient struct {
	client  projectv1.ProjectServiceClient
	timeout time.Duration
}

func NewProjectClient(client projectv1.ProjectServiceClient, timeout time.Duration) *projectClient {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	return &projectClient{client: client, timeout: timeout}
}

// ListDetectable returns the profile's active, non-default projects.
func (c *projectClient) ListDetectable(ctx context.Context, profileID string) ([]model.ProjectCandidate, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.ListProjects(callCtx, &projectv1.ListProjectsRequest{
		ProfileId:       profileID,
		IncludeArchived: false,
	})
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}

	out := make([]model.ProjectCandidate, 0, len(response.GetProjects()))
	for _, project := range response.GetProjects() {
		if project.GetIsDefault() || project.GetIsArchived() {
			continue
		}
		out = append(out, model.ProjectCandidate{
			ID:          project.GetId(),
			Name:        project.GetName(),
			Description: project.GetDescription(),
		})
	}
	return out, nil
}
