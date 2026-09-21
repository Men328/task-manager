package service

import (
	"context"
	"errors"
	"strings"

	"taskmanager/service/identity/internal/model"
)

type authService struct {
	profiles      ProfileRepository
	authProviders AuthProviderRepository
}

func NewAuthService(profiles ProfileRepository, authProviders AuthProviderRepository) AuthService {
	return &authService{profiles: profiles, authProviders: authProviders}
}

func (s *authService) LoginWithProvider(ctx context.Context, in model.ProviderLogin) (model.Profile, bool, error) {
	provider := strings.ToLower(strings.TrimSpace(in.Provider))
	providerUserID := strings.TrimSpace(in.ProviderUserID)
	email := strings.ToLower(strings.TrimSpace(in.Email))

	link, err := s.authProviders.GetByProviderUserID(ctx, provider, providerUserID)
	switch {
	case err == nil:
		return s.resume(ctx, link.ProfileID)
	case errors.Is(err, model.ErrAuthProviderNotFound):
	default:
		return model.Profile{}, false, err
	}

	existing, err := s.profiles.GetByEmail(ctx, email)
	switch {
	case err == nil:
		return s.attach(ctx, existing.ID, provider, providerUserID)
	case errors.Is(err, model.ErrNotFound):
	default:
		return model.Profile{}, false, err
	}

	created, err := s.profiles.Create(ctx, model.Profile{
		Email:       email,
		DisplayName: in.DisplayName,
		AvatarURL:   in.AvatarURL,
		Timezone:    in.Timezone,
		Locale:      in.Locale,
		IsActive:    true,
	})
	if errors.Is(err, model.ErrEmailExists) {
		raced, getErr := s.profiles.GetByEmail(ctx, email)
		if getErr != nil {
			return model.Profile{}, false, getErr
		}
		return s.attach(ctx, raced.ID, provider, providerUserID)
	}
	if err != nil {
		return model.Profile{}, false, err
	}

	if _, err := s.authProviders.Create(ctx, model.AuthProvider{
		ProfileID:      created.ID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}); err != nil {
		return model.Profile{}, false, err
	}

	profile, err := s.profiles.TouchLastLogin(ctx, created.ID)
	if err != nil {
		return model.Profile{}, false, err
	}
	return profile, true, nil
}

func (s *authService) ListProviders(ctx context.Context, profileID string) ([]model.AuthProvider, error) {
	if _, err := s.profiles.Get(ctx, profileID); err != nil {
		return nil, err
	}
	return s.authProviders.ListByProfileID(ctx, profileID)
}

func (s *authService) resume(ctx context.Context, profileID string) (model.Profile, bool, error) {
	profile, err := s.profiles.Get(ctx, profileID)
	if err != nil {
		return model.Profile{}, false, err
	}
	if !profile.IsActive {
		return model.Profile{}, false, model.ErrProfileInactive
	}
	touched, err := s.profiles.TouchLastLogin(ctx, profile.ID)
	if err != nil {
		return model.Profile{}, false, err
	}
	return touched, false, nil
}

func (s *authService) attach(ctx context.Context, profileID string, provider string, providerUserID string) (model.Profile, bool, error) {
	profile, err := s.profiles.Get(ctx, profileID)
	if err != nil {
		return model.Profile{}, false, err
	}
	if !profile.IsActive {
		return model.Profile{}, false, model.ErrProfileInactive
	}
	if _, err := s.authProviders.Create(ctx, model.AuthProvider{
		ProfileID:      profile.ID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}); err != nil {
		return model.Profile{}, false, err
	}
	touched, err := s.profiles.TouchLastLogin(ctx, profile.ID)
	if err != nil {
		return model.Profile{}, false, err
	}
	return touched, false, nil
}
