package repository

import (
	"context"
	"strings"
	"sync"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type InMemorySubscriptionRepository struct {
	mu      sync.RWMutex
	byID    map[string]model.Subscription
	byEmail map[string]string
}

func NewInMemorySubscriptionRepository() *InMemorySubscriptionRepository {
	return &InMemorySubscriptionRepository{
		byID:    map[string]model.Subscription{},
		byEmail: map[string]string{},
	}
}

func (r *InMemorySubscriptionRepository) Upsert(_ context.Context, sub model.Subscription) (model.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if existing, found := r.byID[sub.ProfileID]; found {
		if sub.CreatedAt.IsZero() {
			sub.CreatedAt = existing.CreatedAt
		}
		if sub.Email == "" {
			sub.Email = existing.Email
		}
		if sub.AccessToken == "" {
			sub.AccessToken = existing.AccessToken
			sub.AccessTokenExpiresAt = existing.AccessTokenExpiresAt
		}
		if sub.RefreshToken == "" {
			sub.RefreshToken = existing.RefreshToken
		}
		if sub.HistoryID == "" {
			sub.HistoryID = existing.HistoryID
		}
		if sub.WatchExpiresAt.IsZero() {
			sub.WatchExpiresAt = existing.WatchExpiresAt
		}
	} else if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now

	r.store(sub)
	return sub, nil
}

func (r *InMemorySubscriptionRepository) GetByProfileID(_ context.Context, profileID string) (model.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sub, found := r.byID[profileID]
	if !found {
		return model.Subscription{}, model.ErrNotFound
	}
	return sub, nil
}

func (r *InMemorySubscriptionRepository) GetByEmail(_ context.Context, email string) (model.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	profileID, found := r.byEmail[strings.ToLower(strings.TrimSpace(email))]
	if !found {
		return model.Subscription{}, model.ErrNotFound
	}
	sub, found := r.byID[profileID]
	if !found {
		return model.Subscription{}, model.ErrNotFound
	}
	return sub, nil
}

func (r *InMemorySubscriptionRepository) List(_ context.Context) ([]model.Subscription, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Subscription, 0, len(r.byID))
	for _, sub := range r.byID {
		out = append(out, sub)
	}
	return out, nil
}

func (r *InMemorySubscriptionRepository) UpdateTokens(_ context.Context, profileID string, token model.Token) (model.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, found := r.byID[profileID]
	if !found {
		return model.Subscription{}, model.ErrNotFound
	}
	sub.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		sub.RefreshToken = token.RefreshToken
	}
	sub.AccessTokenExpiresAt = token.ExpiresAt
	sub.UpdatedAt = time.Now().UTC()

	r.store(sub)
	return sub, nil
}

func (r *InMemorySubscriptionRepository) UpdateWatch(_ context.Context, profileID string, historyID string, watchExpiresAt time.Time) (model.Subscription, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, found := r.byID[profileID]
	if !found {
		return model.Subscription{}, model.ErrNotFound
	}
	if historyID != "" {
		sub.HistoryID = historyID
	}
	if !watchExpiresAt.IsZero() {
		sub.WatchExpiresAt = watchExpiresAt
	}
	sub.UpdatedAt = time.Now().UTC()

	r.store(sub)
	return sub, nil
}

func (r *InMemorySubscriptionRepository) Delete(_ context.Context, profileID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	sub, found := r.byID[profileID]
	if !found {
		return model.ErrNotFound
	}
	delete(r.byID, profileID)
	if sub.Email != "" {
		delete(r.byEmail, strings.ToLower(sub.Email))
	}
	return nil
}

func (r *InMemorySubscriptionRepository) store(sub model.Subscription) {
	if existing, found := r.byID[sub.ProfileID]; found && existing.Email != "" && !strings.EqualFold(existing.Email, sub.Email) {
		delete(r.byEmail, strings.ToLower(existing.Email))
	}
	r.byID[sub.ProfileID] = sub
	if sub.Email != "" {
		r.byEmail[strings.ToLower(sub.Email)] = sub.ProfileID
	}
}
