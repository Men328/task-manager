package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"taskmanager/common/errorcode"
	socialv1 "taskmanager/common/gen/go/social/v1"
)

const (
	routeTimeout = 10 * time.Second
	maxBodyBytes = 16 * 1024
)

// friendRoutes là HTTP API của social. Người gọi luôn lấy từ session token,
// không bao giờ từ dữ liệu client gửi lên.
type friendRoutes struct {
	friends socialv1.FriendServiceClient
	signer  *sessionSigner
}

type sendRequestBody struct {
	ProfileID string `json:"profile_id"`
}

func newFriendRoutes(friends socialv1.FriendServiceClient, signer *sessionSigner) *friendRoutes {
	return &friendRoutes{friends: friends, signer: signer}
}

func (a *friendRoutes) register(mux *runtime.ServeMux) error {
	routes := []struct {
		method  string
		path    string
		handler authedHandler
	}{
		{http.MethodGet, "/v1/friends/search", a.handleSearch},
		{http.MethodGet, "/v1/friends", a.handleListFriends},
		{http.MethodDelete, "/v1/friends/{profile_id}", a.handleRemoveFriend},
		{http.MethodGet, "/v1/friend-requests", a.handleListRequests},
		{http.MethodPost, "/v1/friend-requests", a.handleSendRequest},
		{http.MethodPost, "/v1/friend-requests/{id}/accept", a.handleAccept},
		{http.MethodPost, "/v1/friend-requests/{id}/decline", a.handleDecline},
		{http.MethodDelete, "/v1/friend-requests/{id}", a.handleCancel},
	}

	for _, route := range routes {
		if err := mux.HandlePath(route.method, route.path, a.authenticated(route.handler)); err != nil {
			return err
		}
	}
	return nil
}

// authedHandler trả response của gRPC; friendRoutes lo phần xác thực và ghi kết quả.
type authedHandler func(ctx context.Context, r *http.Request, profileID string, params map[string]string) (proto.Message, error)

func (a *friendRoutes) authenticated(handler authedHandler) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, params map[string]string) {
		profileID, ok := a.authenticate(w, r)
		if !ok {
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), routeTimeout)
		defer cancel()

		response, err := handler(ctx, r, profileID, params)
		if err != nil {
			writeAPIError(w, grpcErrorCode(err))
			return
		}
		writeProto(w, response)
	}
}

func (a *friendRoutes) handleSearch(ctx context.Context, r *http.Request, profileID string, _ map[string]string) (proto.Message, error) {
	return a.friends.SearchByEmail(ctx, &socialv1.SearchByEmailRequest{
		ProfileId: profileID,
		Email:     r.URL.Query().Get("email"),
	})
}

func (a *friendRoutes) handleListFriends(ctx context.Context, _ *http.Request, profileID string, _ map[string]string) (proto.Message, error) {
	return a.friends.ListFriends(ctx, &socialv1.ListFriendsRequest{ProfileId: profileID})
}

func (a *friendRoutes) handleRemoveFriend(ctx context.Context, _ *http.Request, profileID string, params map[string]string) (proto.Message, error) {
	return a.friends.RemoveFriend(ctx, &socialv1.RemoveFriendRequest{ProfileId: profileID, FriendId: params["profile_id"]})
}

func (a *friendRoutes) handleListRequests(ctx context.Context, _ *http.Request, profileID string, _ map[string]string) (proto.Message, error) {
	return a.friends.ListFriendRequests(ctx, &socialv1.ListFriendRequestsRequest{ProfileId: profileID})
}

func (a *friendRoutes) handleSendRequest(ctx context.Context, r *http.Request, profileID string, _ map[string]string) (proto.Message, error) {
	targetID, err := parseTargetID(r)
	if err != nil {
		return nil, err
	}
	return a.friends.SendFriendRequest(ctx, &socialv1.SendFriendRequestRequest{ProfileId: profileID, TargetId: targetID})
}

func (a *friendRoutes) handleAccept(ctx context.Context, _ *http.Request, profileID string, params map[string]string) (proto.Message, error) {
	return a.friends.AcceptFriendRequest(ctx, &socialv1.AcceptFriendRequestRequest{ProfileId: profileID, RequestId: params["id"]})
}

func (a *friendRoutes) handleDecline(ctx context.Context, _ *http.Request, profileID string, params map[string]string) (proto.Message, error) {
	return a.friends.DeclineFriendRequest(ctx, &socialv1.DeclineFriendRequestRequest{ProfileId: profileID, RequestId: params["id"]})
}

func (a *friendRoutes) handleCancel(ctx context.Context, _ *http.Request, profileID string, params map[string]string) (proto.Message, error) {
	return a.friends.CancelFriendRequest(ctx, &socialv1.CancelFriendRequestRequest{ProfileId: profileID, RequestId: params["id"]})
}

// parseTargetID đọc người nhận lời mời từ body.
func parseTargetID(r *http.Request) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		return "", errorcode.Error(errorcode.CommonInvalidArgument)
	}
	var body sendRequestBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return "", errorcode.Error(errorcode.CommonInvalidArgument)
	}
	targetID := strings.TrimSpace(body.ProfileID)
	if targetID == "" {
		return "", errorcode.Error(errorcode.CommonInvalidArgument)
	}
	return targetID, nil
}

func (a *friendRoutes) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
	raw := bearerToken(r.Header.Get("Authorization"))
	if raw == "" {
		writeAPIError(w, errorcode.IdentityAuthTokenMissing)
		return "", false
	}

	profileID, err := a.signer.verify(raw)
	if err != nil {
		writeAPIError(w, errorcode.IdentityAuthTokenInvalid)
		return "", false
	}
	return profileID, true
}

func writeProto(w http.ResponseWriter, message proto.Message) {
	encoded, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(message)
	if err != nil {
		writeAPIError(w, errorcode.CommonInternal)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(encoded)
}

func bearerToken(header string) string {
	scheme, value, found := strings.Cut(strings.TrimSpace(header), " ")
	if !found || !strings.EqualFold(scheme, "bearer") {
		return ""
	}
	return strings.TrimSpace(value)
}

func grpcErrorCode(err error) string {
	st := status.Convert(err)
	for _, detail := range st.Details() {
		info, ok := detail.(*errdetails.ErrorInfo)
		if ok && strings.TrimSpace(info.GetReason()) != "" {
			return info.GetReason()
		}
	}

	switch st.Code() {
	case codes.Unauthenticated:
		return errorcode.IdentityAuthTokenInvalid
	case codes.NotFound:
		return errorcode.SocialProfileNotFound
	case codes.InvalidArgument:
		return errorcode.CommonInvalidArgument
	case codes.PermissionDenied:
		return errorcode.CommonFailedPrecondition
	case codes.Unavailable:
		return errorcode.CommonUnavailable
	default:
		return errorcode.CommonInternal
	}
}

func writeAPIError(w http.ResponseWriter, code string) {
	entry, found := errorcode.Lookup(code)
	httpStatus := http.StatusInternalServerError
	if found && entry.HTTPStatus > 0 {
		httpStatus = entry.HTTPStatus
	}
	message := errorcode.DefaultMessage(code)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    code,
		"message": message,
		"details": []map[string]any{{
			"@type":  "type.googleapis.com/google.rpc.ErrorInfo",
			"reason": code,
			"domain": errorcode.Domain,
		}},
	})
}
