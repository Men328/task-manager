package repository

import (
	"context"
	"fmt"
	"time"

	projectv1 "taskmanager/common/gen/go/project/v1"
)

type defaultProjectSeeder struct {
	client  projectv1.ProjectServiceClient
	timeout time.Duration
}

func NewDefaultProjectSeeder(client projectv1.ProjectServiceClient, timeout time.Duration) *defaultProjectSeeder {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &defaultProjectSeeder{client: client, timeout: timeout}
}

func (s *defaultProjectSeeder) SeedDefaultProject(ctx context.Context, profileID string) error {
	callCtx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	if _, err := s.client.SeedDefaultProject(callCtx, &projectv1.SeedDefaultProjectRequest{
		ProfileId: profileID,
	}); err != nil {
		return fmt.Errorf("seed project mặc định cho profile %s: %w", profileID, err)
	}
	return nil
}
