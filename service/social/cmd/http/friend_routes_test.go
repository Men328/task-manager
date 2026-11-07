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
	socialv1 "taskmanager/common/gen/go/social/v1"
)

const testSecret = "test-secret-at-least-16-bytes"

type stubFriendClient struct {
	socialv1.FriendServiceClient
	calls  int
	err    error
	search *socialv1.SearchByEmailRequest
	send   *socialv1.SendFriendRequestRequest
	accept *socialv1.AcceptFriendRequestRequest
	remove *socialv1.RemoveFriendRequest
}

func (c *stubFriendClient) SearchByEmail(_ context.Context, in *socialv1.SearchByEmailRequest, _ ...grpc.CallOption) (*socialv1.SearchByEmailResponse, error) {
	c.calls++
	c.search = in
	return &socialv1.SearchByEmailResponse{Relation: "none"}, c.err
}

func (c *stubFriendClient) ListFriends(context.Context, *socialv1.ListFriendsRequest, ...grpc.CallOption) (*socialv1.ListFriendsResponse, error) {
	c.calls++
	return &socialv1.ListFriendsResponse{}, c.err
}

func (c *stubFriendClient) RemoveFriend(_ context.Context, in *socialv1.RemoveFriendRequest, _ ...grpc.CallOption) (*socialv1.RemoveFriendResponse, error) {
	c.calls++
	c.remove = in
	return &socialv1.RemoveFriendResponse{}, c.err
}

func (c *stubFriendClient) ListFriendRequests(context.Context, *socialv1.ListFriendRequestsRequest, ...grpc.CallOption) (*socialv1.ListFriendRequestsResponse, error) {
	c.calls++
	return &socialv1.ListFriendRequestsResponse{}, c.err
}

func (c *stubFriendClient) SendFriendRequest(_ context.Context, in *socialv1.SendFriendRequestRequest, _ ...grpc.CallOption) (*socialv1.SendFriendRequestResponse, error) {
	c.calls++
	c.send = in
	return &socialv1.SendFriendRequestResponse{Relation: "outgoing"}, c.err
}

func (c *stubFriendClient) AcceptFriendRequest(_ context.Context, in *socialv1.AcceptFriendRequestRequest, _ ...grpc.CallOption) (*socialv1.AcceptFriendRequestResponse, error) {
	c.calls++
	c.accept = in
	return &socialv1.AcceptFriendRequestResponse{}, c.err
}

func (c *stubFriendClient) DeclineFriendRequest(context.Context, *socialv1.DeclineFriendRequestRequest, ...grpc.CallOption) (*socialv1.DeclineFriendRequestResponse, error) {
	c.calls++
	return &socialv1.DeclineFriendRequestResponse{}, c.err
}

func (c *stubFriendClient) CancelFriendRequest(context.Context, *socialv1.CancelFriendRequestRequest, ...grpc.CallOption) (*socialv1.CancelFriendRequestResponse, error) {
	c.calls++
	return &socialv1.CancelFriendRequestResponse{}, c.err
}

func newTestMux(t *testing.T, client socialv1.FriendServiceClient) *runtime.ServeMux {
	t.Helper()
	mux := runtime.NewServeMux()
	routes := newFriendRoutes(client, &sessionSigner{secret: []byte(testSecret)})
	if err := routes.register(mux); err != nil {
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
	{http.MethodGet, "/v1/friends/search?email=a@x.io", ""},
	{http.MethodGet, "/v1/friends", ""},
	{http.MethodDelete, "/v1/friends/p1", ""},
	{http.MethodGet, "/v1/friend-requests", ""},
	{http.MethodPost, "/v1/friend-requests", `{"profile_id":"p1"}`},
	{http.MethodPost, "/v1/friend-requests/r1/accept", ""},
	{http.MethodPost, "/v1/friend-requests/r1/decline", ""},
	{http.MethodDelete, "/v1/friend-requests/r1", ""},
}

func TestRoutesRejectMissingToken(t *testing.T) {
	for _, route := range allRoutes {
		client := &stubFriendClient{}
		recorder := call(newTestMux(t, client), route.method, route.path, "", route.body)

		if recorder.Code != http.StatusUnauthorized || errorCode(t, recorder) != errorcode.IdentityAuthTokenMissing {
			t.Fatalf("%s %s: phải 401 token missing, nhận %d %s", route.method, route.path, recorder.Code, recorder.Body.String())
		}
		if client.calls != 0 {
			t.Fatalf("%s %s: không được gọi gRPC", route.method, route.path)
		}
	}
}

func TestRoutesRejectInvalidToken(t *testing.T) {
	tokens := map[string]string{
		"sai secret": signToken(t, "another-secret-at-least-16", "a", time.Hour),
		"hết hạn":    signToken(t, testSecret, "a", -time.Hour),
	}
	for name, token := range tokens {
		for _, route := range allRoutes {
			client := &stubFriendClient{}
			recorder := call(newTestMux(t, client), route.method, route.path, token, route.body)

			if errorCode(t, recorder) != errorcode.IdentityAuthTokenInvalid || client.calls != 0 {
				t.Fatalf("%s, %s %s: phải từ chối, nhận %d %s", name, route.method, route.path, recorder.Code, recorder.Body.String())
			}
		}
	}
}

func TestRoutesAcceptValidToken(t *testing.T) {
	token := signToken(t, testSecret, "a", time.Hour)
	for _, route := range allRoutes {
		client := &stubFriendClient{}
		recorder := call(newTestMux(t, client), route.method, route.path, token, route.body)

		if recorder.Code != http.StatusOK || client.calls != 1 {
			t.Fatalf("%s %s: phải 200 và gọi 1 lần, nhận %d, %d lần", route.method, route.path, recorder.Code, client.calls)
		}
	}
}

func TestCallerComesFromToken(t *testing.T) {
	client := &stubFriendClient{}
	token := signToken(t, testSecret, "A", time.Hour)

	recorder := call(newTestMux(t, client), http.MethodPost, "/v1/friend-requests", token, `{"profile_id":"B","requester_id":"C"}`)

	if recorder.Code != http.StatusOK {
		t.Fatalf("phải 200, nhận %d %s", recorder.Code, recorder.Body.String())
	}
	if client.send.GetProfileId() != "A" || client.send.GetTargetId() != "B" {
		t.Fatalf("người gọi phải là A, người nhận là B, nhận %+v", client.send)
	}
}

func TestSendRequestRejectsBadBody(t *testing.T) {
	token := signToken(t, testSecret, "A", time.Hour)
	for _, body := range []string{"", "not-json", `{"profile_id":"  "}`} {
		client := &stubFriendClient{}
		recorder := call(newTestMux(t, client), http.MethodPost, "/v1/friend-requests", token, body)

		if recorder.Code != http.StatusBadRequest || client.calls != 0 {
			t.Fatalf("body %q: phải 400, nhận %d", body, recorder.Code)
		}
	}
}

func TestSearchPassesEmailQuery(t *testing.T) {
	client := &stubFriendClient{}
	token := signToken(t, testSecret, "A", time.Hour)

	recorder := call(newTestMux(t, client), http.MethodGet, "/v1/friends/search?email=an%2Bx%40y.io", token, "")

	if client.search.GetProfileId() != "A" || client.search.GetEmail() != "an+x@y.io" {
		t.Fatalf("query sai: %+v", client.search)
	}
	var payload map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatalf("đọc response: %v", err)
	}
	if profile, ok := payload["profile"]; !ok || profile != nil {
		t.Fatalf("không thấy phải trả profile null, nhận %s", recorder.Body.String())
	}
}

func TestAcceptUsesPathId(t *testing.T) {
	client := &stubFriendClient{}
	token := signToken(t, testSecret, "A", time.Hour)

	call(newTestMux(t, client), http.MethodPost, "/v1/friend-requests/r9/accept", token, "")

	if client.accept.GetProfileId() != "A" || client.accept.GetRequestId() != "r9" {
		t.Fatalf("request sai: %+v", client.accept)
	}
}

func TestRemoveFriendUsesPathId(t *testing.T) {
	client := &stubFriendClient{}
	token := signToken(t, testSecret, "A", time.Hour)

	call(newTestMux(t, client), http.MethodDelete, "/v1/friends/p9", token, "")

	if client.remove.GetProfileId() != "A" || client.remove.GetFriendId() != "p9" {
		t.Fatalf("request sai: %+v", client.remove)
	}
}

func TestGRPCErrorMapsToApiError(t *testing.T) {
	token := signToken(t, testSecret, "A", time.Hour)

	client := &stubFriendClient{err: errorcode.Error(errorcode.SocialAlreadyFriends)}
	recorder := call(newTestMux(t, client), http.MethodPost, "/v1/friend-requests", token, `{"profile_id":"B"}`)
	if recorder.Code != http.StatusConflict || errorCode(t, recorder) != errorcode.SocialAlreadyFriends {
		t.Fatalf("phải 409, nhận %d %s", recorder.Code, recorder.Body.String())
	}

	client = &stubFriendClient{err: status.Error(codes.Unavailable, "down")}
	recorder = call(newTestMux(t, client), http.MethodGet, "/v1/friends", token, "")
	if recorder.Code != http.StatusServiceUnavailable || errorCode(t, recorder) != errorcode.CommonUnavailable {
		t.Fatalf("phải 503, nhận %d %s", recorder.Code, recorder.Body.String())
	}
}
