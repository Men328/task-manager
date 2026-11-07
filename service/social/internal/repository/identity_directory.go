package repository

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	identityv1 "taskmanager/common/gen/go/identity/v1"
	"taskmanager/service/social/internal/model"
)

// maxBatchIDs khớp giới hạn của identity.BatchGetProfiles.
const maxBatchIDs = 500

type IdentityDirectory struct {
	client  identityv1.ProfileServiceClient
	timeout time.Duration
}

func NewIdentityDirectory(client identityv1.ProfileServiceClient, timeout time.Duration) *IdentityDirectory {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &IdentityDirectory{client: client, timeout: timeout}
}

func (d *IdentityDirectory) FindByEmail(ctx context.Context, email string) (model.Profile, error) {
	callCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	response, err := d.client.FindProfileByEmail(callCtx, &identityv1.FindProfileByEmailRequest{Email: email})
	if err != nil {
		return model.Profile{}, mapIdentityStatus(err)
	}
	return profileFromProto(response.GetProfile()), nil
}

func (d *IdentityDirectory) GetByIDs(ctx context.Context, ids []string) ([]model.Profile, error) {
	out := make([]model.Profile, 0, len(ids))
	for start := 0; start < len(ids); start += maxBatchIDs {
		end := min(start+maxBatchIDs, len(ids))
		profiles, err := d.getBatch(ctx, ids[start:end])
		if err != nil {
			return nil, err
		}
		out = append(out, profiles...)
	}
	return out, nil
}

func (d *IdentityDirectory) getBatch(ctx context.Context, ids []string) ([]model.Profile, error) {
	callCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	response, err := d.client.BatchGetProfiles(callCtx, &identityv1.BatchGetProfilesRequest{Ids: ids})
	if err != nil {
		return nil, mapIdentityStatus(err)
	}
	out := make([]model.Profile, 0, len(response.GetProfiles()))
	for _, p := range response.GetProfiles() {
		out = append(out, profileFromProto(p))
	}
	return out, nil
}

func profileFromProto(p *identityv1.Profile) model.Profile {
	return model.Profile{
		ID:          p.GetId(),
		Email:       p.GetEmail(),
		DisplayName: p.GetDisplayName(),
		AvatarURL:   p.GetAvatarUrl(),
	}
}

func mapIdentityStatus(err error) error {
	switch status.Code(err) {
	case codes.NotFound:
		return model.ErrNotFound
	case codes.Unavailable, codes.DeadlineExceeded:
		return fmt.Errorf("%w: %v", model.ErrUnavailable, err)
	default:
		return fmt.Errorf("gọi identity: %w", err)
	}
}
