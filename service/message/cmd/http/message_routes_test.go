package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"taskmanager/common/errorcode"
	messagev1 "taskmanager/common/gen/go/message/v1"
	"taskmanager/service/message/internal/config"
)

const testSecret = "test-secret-at-least-16-bytes"

type stubMessageClient struct {
	messagev1.MessageServiceClient
	calls int
	err   error
	send  *messagev1.SendMessageRequest
	list  *messagev1.ListMessagesRequest
	task  *messagev1.SendTaskMessageRequest
	get   *messagev1.GetMessageRequest
}

func (c *stubMessageClient) SendMessage(_ context.Context, in *messagev1.SendMessageRequest, _ ...grpc.CallOption) (*messagev1.SendMessageResponse, error) {
	c.calls++
	c.send = in
	return &messagev1.SendMessageResponse{Message: &messagev1.ChatMessage{Id: "m1", ClientId: in.GetClientId()}}, c.err
}

func (c *stubMessageClient) ListMessages(_ context.Context, in *messagev1.ListMessagesRequest, _ ...grpc.CallOption) (*messagev1.ListMessagesResponse, error) {
	c.calls++
	c.list = in
	return &messagev1.ListMessagesResponse{}, c.err
}

func (c *stubMessageClient) SendTaskMessage(_ context.Context, in *messagev1.SendTaskMessageRequest, _ ...grpc.CallOption) (*messagev1.SendTaskMessageResponse, error) {
	c.calls++
	c.task = in
	return &messagev1.SendTaskMessageResponse{Message: &messagev1.ChatMessage{Id: "m1"}}, c.err
}

func (c *stubMessageClient) GetMessage(_ context.Context, in *messagev1.GetMessageRequest, _ ...grpc.CallOption) (*messagev1.GetMessageResponse, error) {
	c.calls++
	c.get = in
	return &messagev1.GetMessageResponse{Message: &messagev1.ChatMessage{Id: in.GetMessageId()}}, c.err
}

func (c *stubMessageClient) ListConversations(context.Context, *messagev1.ListConversationsRequest, ...grpc.CallOption) (*messagev1.ListConversationsResponse, error) {
	c.calls++
	return &messagev1.ListConversationsResponse{}, c.err
}

func newTestMux(t *testing.T, client messagev1.MessageServiceClient) *runtime.ServeMux {
	t.Helper()
	mux := runtime.NewServeMux()
	if err := newMessageRoutes(client, &sessionSigner{secret: []byte(testSecret)}).register(mux); err != nil {
		t.Fatalf("register: %v", err)
	}
	return mux
}

func signToken(t *testing.T, secret, subject string, expiresIn time.Duration) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    sessionIssuer,
		Subject:   subject,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiresIn)),
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("ký token: %v", err)
	}
	return signed
}

func call(mux http.Handler, method, target, token, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	return recorder
}

func errorCode(t *testing.T, recorder *httptest.ResponseRecorder) string {
	t.Helper()
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("đọc lỗi: %v (%s)", err, recorder.Body.String())
	}
	return payload.Code
}

var allRoutes = []struct{ method, path, body string }{
	{http.MethodGet, "/v1/conversations", ""},
	{http.MethodGet, "/v1/conversations/p1/messages", ""},
	{http.MethodPost, "/v1/conversations/p1/messages", `{"body":"hi"}`},
	{http.MethodPost, "/v1/conversations/p1/tasks", `{"title":"x"}`},
	{http.MethodGet, "/v1/messages/m1", ""},
}

func TestRoutesRejectMissingOrInvalidToken(t *testing.T) {
	tokens := map[string]string{
		"thiếu":      "",
		"sai secret": signToken(t, "another-secret-at-least-16", "a", time.Hour),
		"hết hạn":    signToken(t, testSecret, "a", -time.Hour),
	}
	for name, token := range tokens {
		for _, route := range allRoutes {
			client := &stubMessageClient{}
			recorder := call(newTestMux(t, client), route.method, route.path, token, route.body)

			if recorder.Code != http.StatusUnauthorized || client.calls != 0 {
				t.Fatalf("%s, %s %s: phải 401 và không gọi gRPC, nhận %d", name, route.method, route.path, recorder.Code)
			}
		}
	}
}

func TestRoutesAcceptValidToken(t *testing.T) {
	token := signToken(t, testSecret, "a", time.Hour)
	for _, route := range allRoutes {
		client := &stubMessageClient{}
		recorder := call(newTestMux(t, client), route.method, route.path, token, route.body)

		if recorder.Code != http.StatusOK || client.calls != 1 {
			t.Fatalf("%s %s: phải 200 và gọi 1 lần, nhận %d, %d lần", route.method, route.path, recorder.Code, client.calls)
		}
	}
}

func TestSendTakesSenderFromToken(t *testing.T) {
	client := &stubMessageClient{}
	token := signToken(t, testSecret, "A", time.Hour)

	recorder := call(newTestMux(t, client), http.MethodPost, "/v1/conversations/B/messages", token, `{"body":"chào","client_id":"tmp-1","sender_id":"C"}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("phải 200, nhận %d %s", recorder.Code, recorder.Body.String())
	}
	if client.send.GetProfileId() != "A" || client.send.GetPeerId() != "B" || client.send.GetBody() != "chào" || client.send.GetClientId() != "tmp-1" {
		t.Fatalf("request sai: %+v", client.send)
	}
	var payload struct {
		Message struct {
			ClientID string `json:"clientId"`
		} `json:"message"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil || payload.Message.ClientID != "tmp-1" {
		t.Fatalf("response phải theo camelCase, nhận %s", recorder.Body.String())
	}
}

func TestSendRejectsBadJSON(t *testing.T) {
	client := &stubMessageClient{}
	token := signToken(t, testSecret, "A", time.Hour)
	recorder := call(newTestMux(t, client), http.MethodPost, "/v1/conversations/B/messages", token, "not-json")

	if recorder.Code != http.StatusBadRequest || client.calls != 0 {
		t.Fatalf("phải 400, nhận %d", recorder.Code)
	}
}

func TestListMessagesParsesCursor(t *testing.T) {
	token := signToken(t, testSecret, "A", time.Hour)

	client := &stubMessageClient{}
	call(newTestMux(t, client), http.MethodGet, "/v1/conversations/B/messages?before=2026-11-08T09:30:00.123456Z&limit=20", token, "")
	want := time.Date(2026, 11, 8, 9, 30, 0, 123456000, time.UTC)
	if !client.list.GetBefore().AsTime().Equal(want) || client.list.GetLimit() != 20 || client.list.GetPeerId() != "B" {
		t.Fatalf("query sai: %+v", client.list)
	}

	for _, query := range []string{"before=yesterday", "limit=abc"} {
		client = &stubMessageClient{}
		recorder := call(newTestMux(t, client), http.MethodGet, "/v1/conversations/B/messages?"+query, token, "")
		if recorder.Code != http.StatusBadRequest || client.calls != 0 {
			t.Fatalf("%s: phải 400, nhận %d", query, recorder.Code)
		}
	}
}

func TestGRPCErrorMapsToAPIError(t *testing.T) {
	token := signToken(t, testSecret, "A", time.Hour)

	client := &stubMessageClient{err: errorcode.Error(errorcode.MessageNotFriends)}
	recorder := call(newTestMux(t, client), http.MethodPost, "/v1/conversations/B/messages", token, `{"body":"hi"}`)
	if recorder.Code != http.StatusForbidden || errorCode(t, recorder) != errorcode.MessageNotFriends {
		t.Fatalf("phải 403 not friends, nhận %d %s", recorder.Code, recorder.Body.String())
	}

	client = &stubMessageClient{err: status.Error(codes.Unavailable, "down")}
	recorder = call(newTestMux(t, client), http.MethodGet, "/v1/conversations", token, "")
	if recorder.Code != http.StatusServiceUnavailable || errorCode(t, recorder) != errorcode.CommonUnavailable {
		t.Fatalf("phải 503, nhận %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestUpstreamTimeoutFitsInsideRouteTimeout(t *testing.T) {
	t.Setenv("UPSTREAM_TIMEOUT", "")
	t.Setenv("SOKETI_TIMEOUT", "")

	cfg := config.Load()
	if cfg.UpstreamTimeout+cfg.SoketiTimeout >= routeTimeout {
		t.Fatalf("social %v + soketi %v phải ngắn hơn hạn route %v", cfg.UpstreamTimeout, cfg.SoketiTimeout, routeTimeout)
	}
}

func TestSendTaskParsesBodyAndTakesRequesterFromToken(t *testing.T) {
	client := &stubMessageClient{}
	token := signToken(t, testSecret, "A", time.Hour)

	recorder := call(newTestMux(t, client), http.MethodPost, "/v1/conversations/B/tasks", token,
		`{"title":"Mua sữa","description":"2 hộp","priority":"high","due_at":"2026-11-10T09:00:00Z","client_id":"c1","profile_id":"C"}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("phải 200, nhận %d %s", recorder.Code, recorder.Body.String())
	}
	in := client.task
	if in.GetProfileId() != "A" || in.GetPeerId() != "B" || in.GetTitle() != "Mua sữa" || in.GetPriority() != "high" || in.GetClientId() != "c1" {
		t.Fatalf("request sai: %+v", in)
	}
	if !in.GetDueAt().AsTime().Equal(time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("due_at sai: %v", in.GetDueAt())
	}

	client = &stubMessageClient{}
	recorder = call(newTestMux(t, client), http.MethodPost, "/v1/conversations/B/tasks", token, `{"title":"x","due_at":"mai"}`)
	if recorder.Code != http.StatusBadRequest || client.calls != 0 {
		t.Fatalf("due_at sai định dạng phải 400, nhận %d", recorder.Code)
	}
}

func TestGetMessageUsesCallerFromToken(t *testing.T) {
	client := &stubMessageClient{}
	token := signToken(t, testSecret, "A", time.Hour)

	call(newTestMux(t, client), http.MethodGet, "/v1/messages/m9", token, "")
	if client.get.GetProfileId() != "A" || client.get.GetMessageId() != "m9" {
		t.Fatalf("request sai: %+v", client.get)
	}

	client = &stubMessageClient{err: status.Error(codes.NotFound, "x")}
	recorder := call(newTestMux(t, client), http.MethodGet, "/v1/messages/m9", token, "")
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("phải 404, nhận %d", recorder.Code)
	}
}
