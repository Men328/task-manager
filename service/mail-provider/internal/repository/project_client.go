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

// ListForMail trả các project đang hoạt động của profile: project thường để dò theo
// nội dung email, và project mặc định ("Chung") để chứa email không khớp project nào.
func (c *projectClient) ListForMail(ctx context.Context, profileID string) (model.ProjectSet, error) {
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	response, err := c.client.ListProjects(callCtx, &projectv1.ListProjectsRequest{
		ProfileId:       profileID,
		IncludeArchived: false,
	})
	if err != nil {
		return model.ProjectSet{}, fmt.Errorf("%w: %v", model.ErrProjectList, err)
	}

	out := model.ProjectSet{Detectable: make([]model.ProjectCandidate, 0, len(response.GetProjects()))}
	for _, project := range response.GetProjects() {
		if project.GetIsArchived() {
			continue
		}
		candidate := model.ProjectCandidate{
			ID:          project.GetId(),
			Name:        project.GetName(),
			Description: project.GetDescription(),
		}
		if project.GetIsDefault() {
			if out.Default == nil {
				out.Default = &candidate
			}
			continue
		}
		out.Detectable = append(out.Detectable, candidate)
	}
	return out, nil
}
