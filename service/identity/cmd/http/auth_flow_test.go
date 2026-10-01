package main

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	identityv1 "taskmanager/common/gen/go/identity/v1"
	mailv1 "taskmanager/common/gen/go/mail/v1"
	"taskmanager/service/identity/internal/config"
	"taskmanager/service/identity/internal/handler"
	"taskmanager/service/identity/internal/repository"
	"taskmanager/service/identity/internal/service"
)

type fakeMail struct {
	subscribes []*mailv1.SubscribeRequest
	err        error
}

func (f *fakeMail) Subscribe(_ context.Context, in *mailv1.SubscribeRequest, _ ...grpc.CallOption) (*mailv1.SubscribeResponse, error) {
	f.subscribes = append(f.subscribes, in)
	if f.err != nil {
		return nil, f.err
	}
	return &mailv1.SubscribeResponse{Subscription: &mailv1.Subscription{
		ProfileId: in.GetProfileId(),
		Email:     in.GetEmail(),
	}}, nil
}

func (f *fakeMail) Unsubscribe(_ context.Context, _ *mailv1.UnsubscribeRequest, _ ...grpc.CallOption) (*mailv1.UnsubscribeResponse, error) {
	return &mailv1.UnsubscribeResponse{}, nil
}

func (f *fakeMail) GetSubscription(_ context.Context, _ *mailv1.GetSubscriptionRequest, _ ...grpc.CallOption) (*mailv1.GetSubscriptionResponse, error) {
	return &mailv1.GetSubscriptionResponse{}, nil
}

func (f *fakeMail) HandleNotification(_ context.Context, _ *mailv1.HandleNotificationRequest, _ ...grpc.CallOption) (*mailv1.HandleNotificationResponse, error) {
	return &mailv1.HandleNotificationResponse{}, nil
}

type fakeGoogle struct {
	server    *httptest.Server
	userEmail string
	userSub   string
	verified  bool
	tokenHits int
}

func newFakeGoogle(t *testing.T, sub string, email string) *fakeGoogle {
	t.Helper()

	fake := &fakeGoogle{userSub: sub, userEmail: email, verified: true}
	mux := http.NewServeMux()

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Form.Get("code") == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fake.tokenHits++
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "fake-access-token",
			"refresh_token": "fake-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	})

	mux.HandleFunc("/userinfo", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"sub":            fake.userSub,
			"email":          fake.userEmail,
			"email_verified": fake.verified,
			"name":           "Người Dùng Google",
			"picture":        "https://example.com/avatar.png",
			"locale":         "vi",
		})
	})

	fake.server = httptest.NewServer(mux)
	t.Cleanup(fake.server.Close)
	return fake
}

func newTestAuthEnv(t *testing.T, fake *fakeGoogle) (*authRoutes, *repository.InMemoryProfileRepository) {
	t.Helper()

	profiles := repository.NewInMemoryProfileRepository()
	providers := repository.NewInMemoryAuthProviderRepository()
	authService := service.NewAuthService(profiles, providers, nil)
	profileService := service.NewProfileService(profiles, nil)
	profileHandler := handler.NewProfileHandler(profileService, authService)

	grpcServer := grpc.NewServer()
	identityv1.RegisterProfileServiceServer(grpcServer, profileHandler)

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { _ = grpcServer.Serve(listener) }()
	t.Cleanup(grpcServer.Stop)

	conn, err := grpc.NewClient(listener.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	cfg := config.Config{
		SessionSecret:   testSecret,
		SessionTTL:      time.Hour,
		FrontendBaseURL: "http://fe.test",
	}
	google := newGoogleOAuth(config.Config{
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		GoogleRedirectURL:  "http://localhost:8081/v1/auth/google/callback",
	})
	google.oauth.Endpoint = oauth2.Endpoint{
		AuthURL:  fake.server.URL + "/auth",
		TokenURL: fake.server.URL + "/token",
	}
	googleUserInfoURL = fake.server.URL + "/userinfo"

	routes := newAuthRoutes(cfg, identityv1.NewProfileServiceClient(conn), &fakeMail{}, google, newSessionSigner(cfg))
	return routes, profiles
}

func loginURL(t *testing.T, routes *authRoutes) *url.URL {
	t.Helper()

	recorder := httptest.NewRecorder()
	routes.handleGoogleLogin(recorder, httptest.NewRequest(http.MethodGet, "/v1/auth/google/login", nil), nil)

	if recorder.Code != http.StatusFound {
		t.Fatalf("login phải trả 302, nhận %d (%s)", recorder.Code, recorder.Body.String())
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	return location
}

func callback(t *testing.T, routes *authRoutes, state string) *httptest.ResponseRecorder {
	t.Helper()

	target := "/v1/auth/google/callback?code=fake-code&state=" + url.QueryEscape(state)
	recorder := httptest.NewRecorder()
	routes.handleGoogleCallback(recorder, httptest.NewRequest(http.MethodGet, target, nil), nil)
	return recorder
}

func tokenFromRedirect(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()

	if recorder.Code != http.StatusFound {
		t.Fatalf("callback phải trả 302, nhận %d (%s)", recorder.Code, recorder.Body.String())
	}
	location := recorder.Header().Get("Location")
	if !strings.HasPrefix(location, "http://fe.test/auth/callback#") {
		t.Fatalf("redirect sai đích: %s", location)
	}
	fragment, err := url.ParseQuery(strings.TrimPrefix(location, "http://fe.test/auth/callback#"))
	if err != nil {
		t.Fatalf("parse fragment: %v", err)
	}
	if errCode := fragment.Get("error"); errCode != "" {
		t.Fatalf("callback trả lỗi %s", errCode)
	}
	token := fragment.Get("token")
	if token == "" {
		t.Fatal("redirect thiếu token")
	}
	return token
}

func TestGoogleCallbackCreatesProfileThenReusesIt(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-42", "New.User@Example.com")
	routes, profiles := newTestAuthEnv(t, fake)

	first := loginURL(t, routes)
	if !strings.HasPrefix(first.String(), fake.server.URL+"/auth") {
		t.Fatalf("phải redirect sang Google: %s", first)
	}
	if first.Query().Get("code_challenge") == "" || first.Query().Get("code_challenge_method") != "S256" {
		t.Fatal("authorization URL phải kèm PKCE S256")
	}

	token := tokenFromRedirect(t, callback(t, routes, first.Query().Get("state")))

	claims, err := routes.signer.verify(token)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}

	created, err := profiles.GetByEmail(context.Background(), "new.user@example.com")
	if err != nil {
		t.Fatalf("profile phải được tạo trong DB: %v", err)
	}
	if created.ID != claims.Subject {
		t.Fatalf("token phải trỏ tới profile vừa tạo: %s != %s", claims.Subject, created.ID)
	}
	if created.DisplayName != "Người Dùng Google" {
		t.Fatalf("display_name phải lấy từ Google, nhận %q", created.DisplayName)
	}
	if created.LastLoginAt == nil {
		t.Fatal("login phải set last_login_at")
	}

	token2 := tokenFromRedirect(t, callback(t, routes, loginURL(t, routes).Query().Get("state")))
	claims2, err := routes.signer.verify(token2)
	if err != nil {
		t.Fatalf("verify token lần hai: %v", err)
	}
	if claims2.Subject != claims.Subject {
		t.Fatalf("login lần hai phải trả cùng profile: %s != %s", claims2.Subject, claims.Subject)
	}

	all, err := profiles.List(context.Background(), 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("login hai lần chỉ được có 1 profile, nhận %d", len(all))
	}
}

func TestGoogleCallbackRejectsInvalidState(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-1", "user@example.com")
	routes, profiles := newTestAuthEnv(t, fake)

	recorder := callback(t, routes, "state-gia-mao")
	if recorder.Code != http.StatusFound {
		t.Fatalf("callback phải redirect, nhận %d", recorder.Code)
	}
	if location := recorder.Header().Get("Location"); !strings.Contains(location, "error=IDENTITY_AUTH_STATE_INVALID") {
		t.Fatalf("phải báo state không hợp lệ, nhận %s", location)
	}
	if fake.tokenHits != 0 {
		t.Fatal("state sai thì không được gọi Google token endpoint")
	}

	all, err := profiles.List(context.Background(), 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("state sai thì không được tạo profile, nhận %d", len(all))
	}
}

func TestGoogleCallbackRedirectsWhenProviderCancels(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-1", "user@example.com")
	routes, _ := newTestAuthEnv(t, fake)

	recorder := httptest.NewRecorder()
	routes.handleGoogleCallback(recorder, httptest.NewRequest(
		http.MethodGet,
		"/v1/auth/google/callback?error=access_denied",
		nil,
	), nil)

	if location := recorder.Header().Get("Location"); !strings.Contains(location, "error=IDENTITY_AUTH_EXCHANGE_FAILED") {
		t.Fatalf("phải báo lỗi trao đổi token, nhận %s", location)
	}
}

func TestGoogleCallbackRejectsUnverifiedEmail(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-1", "user@example.com")
	fake.verified = false
	routes, profiles := newTestAuthEnv(t, fake)

	recorder := callback(t, routes, loginURL(t, routes).Query().Get("state"))
	if location := recorder.Header().Get("Location"); !strings.Contains(location, "error=IDENTITY_AUTH_EXCHANGE_FAILED") {
		t.Fatalf("email chưa xác minh phải bị từ chối, nhận %s", location)
	}

	all, err := profiles.List(context.Background(), 50)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 0 {
		t.Fatalf("không được tạo profile, nhận %d", len(all))
	}
}

func TestMeReturnsProfileFromToken(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-7", "me@example.com")
	routes, _ := newTestAuthEnv(t, fake)

	token := tokenFromRedirect(t, callback(t, routes, loginURL(t, routes).Query().Get("state")))

	request := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	recorder := httptest.NewRecorder()
	routes.handleMe(recorder, request, nil)

	if recorder.Code != http.StatusOK {
		t.Fatalf("/v1/auth/me phải trả 200, nhận %d (%s)", recorder.Code, recorder.Body.String())
	}

	var payload struct {
		Profile struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		} `json:"profile"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if payload.Profile.Email != "me@example.com" {
		t.Fatalf("email sai: %q", payload.Profile.Email)
	}

	claims, err := routes.signer.verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if payload.Profile.ID != claims.Subject {
		t.Fatalf("id sai: %q", payload.Profile.ID)
	}
}

func TestMeRejectsMissingAndInvalidToken(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-7", "me@example.com")
	routes, _ := newTestAuthEnv(t, fake)

	recorder := httptest.NewRecorder()
	routes.handleMe(recorder, httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil), nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("thiếu token phải trả 401, nhận %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "IDENTITY_AUTH_TOKEN_MISSING") {
		t.Fatalf("phải trả mã lỗi chuẩn, nhận %s", recorder.Body.String())
	}

	request := httptest.NewRequest(http.MethodGet, "/v1/auth/me", nil)
	request.Header.Set("Authorization", "Bearer token-gia")
	recorder = httptest.NewRecorder()
	routes.handleMe(recorder, request, nil)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("token sai phải trả 401, nhận %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "IDENTITY_AUTH_TOKEN_INVALID") {
		t.Fatalf("phải trả mã lỗi chuẩn, nhận %s", recorder.Body.String())
	}
}

func gmailLoginURL(t *testing.T, routes *authRoutes) *url.URL {
	t.Helper()

	recorder := httptest.NewRecorder()
	routes.handleGoogleLogin(recorder, httptest.NewRequest(http.MethodGet, "/v1/auth/google/login?gmail=1", nil), nil)

	if recorder.Code != http.StatusFound {
		t.Fatalf("login phải trả 302, nhận %d (%s)", recorder.Code, recorder.Body.String())
	}
	location, err := url.Parse(recorder.Header().Get("Location"))
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	return location
}

func TestGoogleLoginWithGmailRequestsOfflineMailScope(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-mail", "mail@example.com")
	routes, _ := newTestAuthEnv(t, fake)

	location := gmailLoginURL(t, routes)
	if scope := location.Query().Get("scope"); !strings.Contains(scope, "gmail.readonly") {
		t.Fatalf("scope phải chứa gmail.readonly, nhận %q", scope)
	}
	if location.Query().Get("access_type") != "offline" {
		t.Fatalf("phải xin access_type=offline, nhận %q", location.Query().Get("access_type"))
	}
	if location.Query().Get("prompt") != "consent" {
		t.Fatalf("phải dùng prompt=consent để có refresh token, nhận %q", location.Query().Get("prompt"))
	}
}

func TestGoogleLoginWithoutGmailKeepsBasicScopes(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-basic", "basic@example.com")
	routes, _ := newTestAuthEnv(t, fake)

	location := loginURL(t, routes)
	if scope := location.Query().Get("scope"); strings.Contains(scope, "gmail.readonly") {
		t.Fatalf("không tích gmail thì không được xin scope gmail, nhận %q", scope)
	}
	if location.Query().Get("access_type") != "" {
		t.Fatalf("không tích gmail thì không cần offline access, nhận %q", location.Query().Get("access_type"))
	}
}

func TestGoogleCallbackSubscribesMailboxWhenGmailRequested(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-mail", "Mail.User@Example.com")
	routes, _ := newTestAuthEnv(t, fake)

	location := gmailLoginURL(t, routes)
	recorder := callback(t, routes, location.Query().Get("state"))

	fragment, err := url.ParseQuery(strings.TrimPrefix(recorder.Header().Get("Location"), "http://fe.test/auth/callback#"))
	if err != nil {
		t.Fatalf("parse fragment: %v", err)
	}
	if fragment.Get("mail") != "ok" {
		t.Fatalf("redirect phải báo mail=ok, nhận %q", fragment.Get("mail"))
	}

	mail := routes.mail.(*fakeMail)
	if len(mail.subscribes) != 1 {
		t.Fatalf("phải gọi Subscribe đúng 1 lần, nhận %d", len(mail.subscribes))
	}
	request := mail.subscribes[0]
	if request.GetEmail() != "mail.user@example.com" {
		t.Fatalf("email phải được chuẩn hoá chữ thường, nhận %q", request.GetEmail())
	}
	if request.GetRefreshToken() != "fake-refresh-token" {
		t.Fatalf("phải chuyển refresh token cho mail-provider, nhận %q", request.GetRefreshToken())
	}
	if request.GetAccessToken() != "fake-access-token" {
		t.Fatalf("phải chuyển access token cho mail-provider, nhận %q", request.GetAccessToken())
	}
	if request.GetProfileId() == "" {
		t.Fatal("phải chuyển profile_id cho mail-provider")
	}
	if request.GetAccessTokenExpiresAt() == nil {
		t.Fatal("phải chuyển thời điểm hết hạn access token")
	}
}

func TestGoogleCallbackKeepsLoginWhenMailSubscriptionFails(t *testing.T) {
	fake := newFakeGoogle(t, "google-sub-mail", "mail@example.com")
	routes, _ := newTestAuthEnv(t, fake)
	routes.mail.(*fakeMail).err = errors.New("mail provider offline")

	location := gmailLoginURL(t, routes)
	recorder := callback(t, routes, location.Query().Get("state"))

	fragment, err := url.ParseQuery(strings.TrimPrefix(recorder.Header().Get("Location"), "http://fe.test/auth/callback#"))
	if err != nil {
		t.Fatalf("parse fragment: %v", err)
	}
	if fragment.Get("token") == "" {
		t.Fatal("đăng ký mail lỗi vẫn phải đăng nhập được")
	}
	if fragment.Get("mail") != "error" {
		t.Fatalf("redirect phải báo mail=error, nhận %q", fragment.Get("mail"))
	}
}
