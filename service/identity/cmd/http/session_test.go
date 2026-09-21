package main

import (
	"testing"
	"time"

	"taskmanager/service/identity/internal/config"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func newTestSigner() *sessionSigner {
	return newSessionSigner(config.Config{SessionSecret: testSecret, SessionTTL: time.Hour})
}

func TestSessionSignerRoundTrip(t *testing.T) {
	signer := newTestSigner()
	now := time.Now()

	token, expiresAt, err := signer.issue("profile-1", "user@example.com", now)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	if !expiresAt.After(now) {
		t.Fatalf("expiresAt phải ở tương lai, nhận %v", expiresAt)
	}

	claims, err := signer.verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.Subject != "profile-1" {
		t.Fatalf("subject sai: %q", claims.Subject)
	}
	if claims.Email != "user@example.com" {
		t.Fatalf("email sai: %q", claims.Email)
	}
}

func TestSessionSignerRejectsForgedAndExpiredTokens(t *testing.T) {
	signer := newTestSigner()
	token, _, err := signer.issue("profile-1", "user@example.com", time.Now())
	if err != nil {
		t.Fatalf("issue: %v", err)
	}

	if _, err := signer.verify(token + "x"); err == nil {
		t.Fatal("token bị sửa phải bị từ chối")
	}

	other := newSessionSigner(config.Config{SessionSecret: "ffffffffffffffffffffffffffffffff", SessionTTL: time.Hour})
	if _, err := other.verify(token); err == nil {
		t.Fatal("token ký bằng secret khác phải bị từ chối")
	}

	expired, _, err := signer.issue("profile-1", "user@example.com", time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Fatalf("issue token hết hạn: %v", err)
	}
	if _, err := signer.verify(expired); err == nil {
		t.Fatal("token hết hạn phải bị từ chối")
	}
}

func TestSessionSignerWithoutSecret(t *testing.T) {
	signer := newSessionSigner(config.Config{SessionTTL: time.Hour})
	if signer.configured() {
		t.Fatal("thiếu secret thì không được coi là đã cấu hình")
	}
	if _, _, err := signer.issue("profile-1", "user@example.com", time.Now()); err == nil {
		t.Fatal("thiếu secret thì issue phải lỗi")
	}
}

func TestOAuthStateRoundTrip(t *testing.T) {
	signer := newTestSigner()
	raw, err := signer.signState(oauthState{
		Nonce:     "nonce-1",
		Verifier:  "verifier-1",
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("signState: %v", err)
	}

	state, err := signer.verifyState(raw)
	if err != nil {
		t.Fatalf("verifyState: %v", err)
	}
	if state.Nonce != "nonce-1" || state.Verifier != "verifier-1" {
		t.Fatalf("state không khớp: %+v", state)
	}
}

func TestOAuthStateRejectsTampering(t *testing.T) {
	signer := newTestSigner()
	raw, err := signer.signState(oauthState{
		Nonce:     "nonce-1",
		Verifier:  "verifier-1",
		ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("signState: %v", err)
	}

	cases := []string{
		"",
		"khong-co-dau-cham",
		"aaaa.bbbb",
		raw + "x",
		raw[:len(raw)-2],
	}
	for _, candidate := range cases {
		if _, err := signer.verifyState(candidate); err == nil {
			t.Fatalf("state %q phải bị từ chối", candidate)
		}
	}
}

func TestOAuthStateRejectsExpired(t *testing.T) {
	signer := newTestSigner()
	raw, err := signer.signState(oauthState{
		Nonce:     "nonce-1",
		Verifier:  "verifier-1",
		ExpiresAt: time.Now().Add(-time.Minute).Unix(),
	})
	if err != nil {
		t.Fatalf("signState: %v", err)
	}
	if _, err := signer.verifyState(raw); err == nil {
		t.Fatal("state hết hạn phải bị từ chối")
	}
}

func TestBearerToken(t *testing.T) {
	cases := map[string]string{
		"Bearer abc.def":   "abc.def",
		"bearer abc.def":   "abc.def",
		"BEARER abc":       "abc",
		"Bearer":           "",
		"":                 "",
		"Basic abc":        "",
		"Bearer  spaced  ": "spaced",
	}
	for header, want := range cases {
		if got := bearerToken(header); got != want {
			t.Fatalf("bearerToken(%q) = %q, mong đợi %q", header, got, want)
		}
	}
}

func TestCallbackURLKeepsTokenInFragment(t *testing.T) {
	routes := &authRoutes{frontendURL: "http://localhost:5173"}

	tokenURL := routes.callbackURL("token", "abc.def")
	if tokenURL != "http://localhost:5173/auth/callback#token=abc.def" {
		t.Fatalf("URL token sai: %s", tokenURL)
	}

	errorURL := routes.callbackURL("error", "IDENTITY_AUTH_STATE_INVALID")
	if errorURL != "http://localhost:5173/auth/callback#error=IDENTITY_AUTH_STATE_INVALID" {
		t.Fatalf("URL error sai: %s", errorURL)
	}
}
