package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

func TestTokenRefresherExchangesRefreshToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
			return
		}
		if got := r.Form.Get("grant_type"); got != "refresh_token" {
			t.Errorf("grant_type sai: %q", got)
		}
		if got := r.Form.Get("refresh_token"); got != "r1" {
			t.Errorf("refresh_token sai: %q", got)
		}
		if got := r.Form.Get("client_id"); got != "cid" {
			t.Errorf("client_id sai: %q", got)
		}
		if got := r.Form.Get("client_secret"); got != "secret" {
			t.Errorf("client_secret sai: %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "a2",
			"refresh_token": "r2",
			"expires_in":    1800,
		})
	}))
	defer server.Close()

	refresher := NewTokenRefresher("cid", "secret", server.URL, 5*time.Second)
	token, err := refresher.Refresh(context.Background(), "r1")
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	if token.AccessToken != "a2" || token.RefreshToken != "r2" {
		t.Fatalf("token sai: %+v", token)
	}
	if !token.ExpiresAt.After(time.Now()) {
		t.Fatalf("expires_at phải ở tương lai, nhận %s", token.ExpiresAt)
	}
}

func TestTokenRefresherReportsGoogleError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "token đã bị thu hồi",
		})
	}))
	defer server.Close()

	refresher := NewTokenRefresher("cid", "secret", server.URL, 5*time.Second)
	if _, err := refresher.Refresh(context.Background(), "r1"); !errors.Is(err, model.ErrTokenRefresh) {
		t.Fatalf("invalid_grant phải là ErrTokenRefresh, nhận %v", err)
	}
}

func TestTokenRefresherRequiresAccessToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"expires_in": 3600})
	}))
	defer server.Close()

	refresher := NewTokenRefresher("cid", "secret", server.URL, 5*time.Second)
	if _, err := refresher.Refresh(context.Background(), "r1"); !errors.Is(err, model.ErrTokenRefresh) {
		t.Fatalf("thiếu access_token phải là ErrTokenRefresh, nhận %v", err)
	}
}
