package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"taskmanager/common/errorcode"
	"taskmanager/service/identity/internal/config"
)

func newSoketiTestAuth(t *testing.T, appKey string, appSecret string) (*soketiAuth, string) {
	t.Helper()

	cfg := config.Config{
		SessionSecret:       testSecret,
		SessionTTL:          time.Hour,
		SoketiAppKey:        appKey,
		SoketiAppSecret:     appSecret,
		SoketiChannelPrefix: "noti-internal-",
	}
	signer := newSessionSigner(cfg)
	token, _, err := signer.issue("profile-1", "user@example.com", time.Now())
	if err != nil {
		t.Fatalf("phát session token: %v", err)
	}
	return newSoketiAuth(cfg, signer), token
}

func soketiAuthRequestFor(t *testing.T, token string, socketID string, channel string) *http.Request {
	t.Helper()

	form := url.Values{}
	form.Set("socket_id", socketID)
	form.Set("channel_name", channel)

	request := httptest.NewRequest(http.MethodPost, "/v1/auth/soketi", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return request
}

func errorReason(t *testing.T, body []byte) string {
	t.Helper()

	var payload struct {
		Details []struct {
			Reason string `json:"reason"`
		} `json:"details"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("decode lỗi: %v (%s)", err, string(body))
	}
	if len(payload.Details) == 0 {
		return ""
	}
	return payload.Details[0].Reason
}

func TestSoketiAuthSignsOwnChannel(t *testing.T) {
	auth, token := newSoketiTestAuth(t, "app-key", "app-secret")
	channel := "private-noti-internal-profile-1"
	recorder := httptest.NewRecorder()

	auth.handle(recorder, soketiAuthRequestFor(t, token, "123.456", channel), nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, muốn 200 (%s)", recorder.Code, recorder.Body.String())
	}

	var payload map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode body: %v", err)
	}

	mac := hmac.New(sha256.New, []byte("app-secret"))
	mac.Write([]byte("123.456:" + channel))
	want := "app-key:" + hex.EncodeToString(mac.Sum(nil))
	if payload["auth"] != want {
		t.Fatalf("auth = %q, muốn %q", payload["auth"], want)
	}
}

func TestSoketiAuthRejectsOtherUserChannel(t *testing.T) {
	auth, token := newSoketiTestAuth(t, "app-key", "app-secret")
	recorder := httptest.NewRecorder()

	auth.handle(recorder, soketiAuthRequestFor(t, token, "123.456", "private-noti-internal-profile-2"), nil)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, muốn 403", recorder.Code)
	}
	if reason := errorReason(t, recorder.Body.Bytes()); reason != errorcode.IdentityAuthSoketiChannelForbidden {
		t.Fatalf("reason = %q, muốn %q", reason, errorcode.IdentityAuthSoketiChannelForbidden)
	}
}

func TestSoketiAuthRequiresToken(t *testing.T) {
	auth, _ := newSoketiTestAuth(t, "app-key", "app-secret")
	recorder := httptest.NewRecorder()

	auth.handle(recorder, soketiAuthRequestFor(t, "", "123.456", "private-noti-internal-profile-1"), nil)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, muốn 401", recorder.Code)
	}
	if reason := errorReason(t, recorder.Body.Bytes()); reason != errorcode.IdentityAuthTokenMissing {
		t.Fatalf("reason = %q, muốn %q", reason, errorcode.IdentityAuthTokenMissing)
	}
}

func TestSoketiAuthFailsClosedWhenNotConfigured(t *testing.T) {
	auth, token := newSoketiTestAuth(t, "", "")
	recorder := httptest.NewRecorder()

	auth.handle(recorder, soketiAuthRequestFor(t, token, "123.456", "private-noti-internal-profile-1"), nil)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, muốn 500", recorder.Code)
	}
	if reason := errorReason(t, recorder.Body.Bytes()); reason != errorcode.IdentityAuthSoketiNotConfigured {
		t.Fatalf("reason = %q, muốn %q", reason, errorcode.IdentityAuthSoketiNotConfigured)
	}
}

func TestSoketiAuthRejectsInvalidRequest(t *testing.T) {
	auth, token := newSoketiTestAuth(t, "app-key", "app-secret")
	recorder := httptest.NewRecorder()

	auth.handle(recorder, soketiAuthRequestFor(t, token, "", ""), nil)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, muốn 400", recorder.Code)
	}
	if reason := errorReason(t, recorder.Body.Bytes()); reason != errorcode.IdentityAuthSoketiRequestInvalid {
		t.Fatalf("reason = %q, muốn %q", reason, errorcode.IdentityAuthSoketiRequestInvalid)
	}
}
