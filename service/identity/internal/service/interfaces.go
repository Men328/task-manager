package service

import (
	"context"

	"taskmanager/service/identity/internal/model"
)

type ProfileRepository interface {
	Create(ctx context.Context, p model.Profile) (model.Profile, error)
	Get(ctx context.Context, id string) (model.Profile, error)
	GetByEmail(ctx context.Context, email string) (model.Profile, error)
	List(ctx context.Context, limit int) ([]model.Profile, error)
	Update(ctx context.Context, id string, upd model.ProfileUpdate) (model.Profile, error)
	TouchLastLogin(ctx context.Context, id string) (model.Profile, error)
	Delete(ctx context.Context, id string) error
}

type AuthProviderRepository interface {
	Create(ctx context.Context, ap model.AuthProvider) (model.AuthProvider, error)
	GetByProviderUserID(ctx context.Context, provider string, providerUserID string) (model.AuthProvider, error)
	ListByProfileID(ctx context.Context, profileID string) ([]model.AuthProvider, error)
}

type ProfileService interface {
	Create(ctx context.Context, p model.Profile) (model.Profile, error)
	Get(ctx context.Context, id string) (model.Profile, error)
	List(ctx context.Context, limit int) ([]model.Profile, error)
	Update(ctx context.Context, id string, upd model.ProfileUpdate) (model.Profile, error)
	Delete(ctx context.Context, id string) error
}

type AuthService interface {
	LoginWithProvider(ctx context.Context, in model.ProviderLogin) (model.Profile, bool, error)
	ListProviders(ctx context.Context, profileID string) ([]model.AuthProvider, error)
}
