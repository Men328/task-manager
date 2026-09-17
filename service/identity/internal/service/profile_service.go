package service

import (
	"context"
	"strings"

	"taskmanager/service/identity/internal/model"
)

const defaultListLimit = 50

type profileService struct {
	profiles ProfileRepository
}

func NewProfileService(profiles ProfileRepository) ProfileService {
	return &profileService{profiles: profiles}
}

func (s *profileService) Create(ctx context.Context, p model.Profile) (model.Profile, error) {
	p.Email = strings.ToLower(strings.TrimSpace(p.Email))
	return s.profiles.Create(ctx, p)
}

func (s *profileService) Get(ctx context.Context, id string) (model.Profile, error) {
	return s.profiles.Get(ctx, id)
}

func (s *profileService) List(ctx context.Context, limit int) ([]model.Profile, error) {
	if limit <= 0 {
		limit = defaultListLimit
	}
	return s.profiles.List(ctx, limit)
}

func (s *profileService) Update(ctx context.Context, id string, upd model.ProfileUpdate) (model.Profile, error) {
	return s.profiles.Update(ctx, id, upd)
}

func (s *profileService) Delete(ctx context.Context, id string) error {
	return s.profiles.Delete(ctx, id)
}
