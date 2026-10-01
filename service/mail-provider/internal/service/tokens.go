package service

import (
	"context"
	"sync"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

const tokenSkew = 2 * time.Minute

type TokenManager struct {
	subs      SubscriptionRepository
	refresher TokenRefresher
	mu        sync.Mutex
	locks     map[string]*sync.Mutex
}

func NewTokenManager(subs SubscriptionRepository, refresher TokenRefresher) *TokenManager {
	return &TokenManager{subs: subs, refresher: refresher, locks: map[string]*sync.Mutex{}}
}

func (m *TokenManager) AccessToken(ctx context.Context, sub model.Subscription) (string, error) {
	if valid(sub.AccessToken, sub.AccessTokenExpiresAt) {
		return sub.AccessToken, nil
	}

	lock := m.lockFor(sub.ProfileID)
	lock.Lock()
	defer lock.Unlock()

	current, err := m.subs.GetByProfileID(ctx, sub.ProfileID)
	if err == nil && valid(current.AccessToken, current.AccessTokenExpiresAt) {
		return current.AccessToken, nil
	}

	refreshToken := sub.RefreshToken
	if refreshToken == "" && err == nil {
		refreshToken = current.RefreshToken
	}
	if refreshToken == "" || m.refresher == nil {
		if sub.AccessToken != "" {
			return sub.AccessToken, nil
		}
		return "", model.ErrTokenRefresh
	}

	token, refreshErr := m.refresher.Refresh(ctx, refreshToken)
	if refreshErr != nil {
		return "", refreshErr
	}
	if token.RefreshToken == "" {
		token.RefreshToken = refreshToken
	}
	if token.ExpiresAt.IsZero() {
		token.ExpiresAt = time.Now().Add(time.Hour)
	}
	if _, updateErr := m.subs.UpdateTokens(ctx, sub.ProfileID, token); updateErr != nil {
		return "", updateErr
	}
	return token.AccessToken, nil
}

func (m *TokenManager) lockFor(profileID string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	lock, found := m.locks[profileID]
	if !found {
		lock = &sync.Mutex{}
		m.locks[profileID] = lock
	}
	return lock
}

func valid(accessToken string, expiresAt time.Time) bool {
	return accessToken != "" && time.Now().Add(tokenSkew).Before(expiresAt)
}
