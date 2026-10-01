package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

func TestTokenManagerUsesCachedToken(t *testing.T) {
	subs := newStubSubscriptions(model.Subscription{
		ProfileID:            "p1",
		AccessToken:          "cached",
		AccessTokenExpiresAt: time.Now().Add(time.Hour),
	})
	refresher := &stubRefresher{token: model.Token{AccessToken: "new", ExpiresAt: time.Now().Add(time.Hour)}}
	manager := NewTokenManager(subs, refresher)

	token, err := manager.AccessToken(context.Background(), mustSubscription(t, subs, "p1"))
	if err != nil {
		t.Fatalf("access token: %v", err)
	}
	if token != "cached" {
		t.Fatalf("phải dùng token còn hạn, nhận %q", token)
	}
	if refresher.calls != 0 {
		t.Fatalf("không được refresh khi token còn hạn, số lần gọi %d", refresher.calls)
	}
}

func TestTokenManagerRefreshesExpiredToken(t *testing.T) {
	subs := newStubSubscriptions(model.Subscription{
		ProfileID:            "p1",
		AccessToken:          "expired",
		RefreshToken:         "refresh",
		AccessTokenExpiresAt: time.Now().Add(-time.Minute),
	})
	refresher := &stubRefresher{token: model.Token{AccessToken: "fresh", ExpiresAt: time.Now().Add(time.Hour)}}
	manager := NewTokenManager(subs, refresher)

	token, err := manager.AccessToken(context.Background(), mustSubscription(t, subs, "p1"))
	if err != nil {
		t.Fatalf("access token: %v", err)
	}
	if token != "fresh" {
		t.Fatalf("phải dùng token mới, nhận %q", token)
	}

	stored, err := subs.GetByProfileID(context.Background(), "p1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if stored.AccessToken != "fresh" {
		t.Fatalf("token mới phải được lưu lại, nhận %q", stored.AccessToken)
	}
	if stored.RefreshToken != "refresh" {
		t.Fatalf("refresh token phải được giữ, nhận %q", stored.RefreshToken)
	}
}

func TestTokenManagerPropagatesRefreshFailure(t *testing.T) {
	subs := newStubSubscriptions(model.Subscription{
		ProfileID:            "p1",
		AccessToken:          "expired",
		RefreshToken:         "refresh",
		AccessTokenExpiresAt: time.Now().Add(-time.Minute),
	})
	refresher := &stubRefresher{err: errors.New("google từ chối")}
	manager := NewTokenManager(subs, refresher)

	if _, err := manager.AccessToken(context.Background(), mustSubscription(t, subs, "p1")); err == nil {
		t.Fatal("refresh lỗi phải trả lỗi")
	}
}

func mustSubscription(t *testing.T, subs *stubSubscriptions, profileID string) model.Subscription {
	t.Helper()
	sub, err := subs.GetByProfileID(context.Background(), profileID)
	if err != nil {
		t.Fatalf("get subscription: %v", err)
	}
	return sub
}
