package repository

import (
	"context"
	"fmt"
	"time"

	taskv1 "taskmanager/common/gen/go/task/v1"
)

type taskLifecycleSeeder struct {
	client  taskv1.TaskStatusServiceClient
	timeout time.Duration
}

func NewTaskLifecycleSeeder(client taskv1.TaskStatusServiceClient, timeout time.Duration) *taskLifecycleSeeder {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &taskLifecycleSeeder{client: client, timeout: timeout}
}

func (s *taskLifecycleSeeder) SeedDefaultStatuses(ctx context.Context, profileID string) error {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if _, err := s.client.SeedDefaultStatuses(callCtx, &taskv1.SeedDefaultStatusesRequest{
		ProfileId: profileID,
	}); err != nil {
		return fmt.Errorf("seed lifecycle mặc định cho profile %s: %w", profileID, err)
	}
	return nil
}
