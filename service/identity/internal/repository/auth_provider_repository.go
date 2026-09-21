package repository

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/identity/internal/model"
)

type InMemoryAuthProviderRepository struct {
	mu    sync.RWMutex
	items map[string]model.AuthProvider
}

func NewInMemoryAuthProviderRepository() *InMemoryAuthProviderRepository {
	return &InMemoryAuthProviderRepository{items: make(map[string]model.AuthProvider)}
}

func (r *InMemoryAuthProviderRepository) Create(_ context.Context, ap model.AuthProvider) (model.AuthProvider, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.items {
		if existing.Provider == ap.Provider && existing.ProviderUserID == ap.ProviderUserID {
			ap.ID = existing.ID
			ap.CreatedAt = existing.CreatedAt
			r.items[existing.ID] = ap
			return ap, nil
		}
	}

	if ap.ID == "" {
		ap.ID = uuid.NewString()
	}
	ap.CreatedAt = time.Now().UTC()
	r.items[ap.ID] = ap
	return ap, nil
}

func (r *InMemoryAuthProviderRepository) GetByProviderUserID(_ context.Context, provider string, providerUserID string) (model.AuthProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, ap := range r.items {
		if ap.Provider == provider && ap.ProviderUserID == providerUserID {
			return ap, nil
		}
	}
	return model.AuthProvider{}, model.ErrAuthProviderNotFound
}

func (r *InMemoryAuthProviderRepository) ListByProfileID(_ context.Context, profileID string) ([]model.AuthProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.AuthProvider, 0, len(r.items))
	for _, ap := range r.items {
		if ap.ProfileID == profileID {
			out = append(out, ap)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}
