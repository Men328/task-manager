package service

import (
	"context"

	"taskmanager/service/identity/internal/model"
)

type ProfileRepository interface {
	Create(ctx context.Context, p model.Profile) (model.Profile, error)
	Get(ctx context.Context, id string) (model.Profile, error)
	List(ctx context.Context, limit int) ([]model.Profile, error)
	Update(ctx context.Context, id string, upd model.ProfileUpdate) (model.Profile, error)
	Delete(ctx context.Context, id string) error
}

type ProfileService interface {
	Create(ctx context.Context, p model.Profile) (model.Profile, error)
	Get(ctx context.Context, id string) (model.Profile, error)
	List(ctx context.Context, limit int) ([]model.Profile, error)
	Update(ctx context.Context, id string, upd model.ProfileUpdate) (model.Profile, error)
	Delete(ctx context.Context, id string) error
}
