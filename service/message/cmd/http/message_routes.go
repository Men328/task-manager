package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	messagev1 "taskmanager/common/gen/go/message/v1"
)

const (
	routeTimeout = 10 * time.Second
	maxBodyBytes = 32 * 1024
)

type messageRoutes struct {
	messages messagev1.MessageServiceClient
	signer   *sessionSigner
}

type sendMessageBody struct {
	Body     string `json:"body"`
	ClientID string `json:"client_id"`
}

type sendTaskBody struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	DueAt       string `json:"due_at"`
	ClientID    string `json:"client_id"`
}

func newMessageRoutes(messages messagev1.MessageServiceClient, signer *sessionSigner) *messageRoutes {
	return &messageRoutes{messages: messages, signer: signer}
}

func (a *messageRoutes) register(mux *runtime.ServeMux) error {
	routes := []struct {
		method  string
		path    string
		handler authedHandler
	}{
		{http.MethodGet, "/v1/conversations", a.handleListConversations},
		{http.MethodGet, "/v1/conversations/{peer_id}/messages", a.handleListMessages},
		{http.MethodPost, "/v1/conversations/{peer_id}/messages", a.handleSend},
		{http.MethodPost, "/v1/conversations/{peer_id}/tasks", a.handleSendTask},
		{http.MethodGet, "/v1/messages/{message_id}", a.handleGetMessage},
	}

	for _, route := range routes {
		if err := mux.HandlePath(route.method, route.path, a.authenticated(route.handler)); err != nil {
			return err
		}
	}
	return nil
}

type authedHandler func(ctx context.Context, r *http.Request, profileID string, params map[string]string) (proto.Message, error)

func (a *messageRoutes) authenticated(handler authedHandler) runtime.HandlerFunc {
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

func (a *messageRoutes) handleListConversations(ctx context.Context, _ *http.Request, profileID string, _ map[string]string) (proto.Message, error) {
	return a.messages.ListConversations(ctx, &messagev1.ListConversationsRequest{ProfileId: profileID})
}

func (a *messageRoutes) handleListMessages(ctx context.Context, r *http.Request, profileID string, params map[string]string) (proto.Message, error) {
	query := r.URL.Query()
	request := &messagev1.ListMessagesRequest{ProfileId: profileID, PeerId: params["peer_id"]}

	if raw := strings.TrimSpace(query.Get("before")); raw != "" {
		before, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, errorcode.Error(errorcode.CommonInvalidArgument)
		}
		request.Before = timestamppb.New(before)
	}
	if raw := strings.TrimSpace(query.Get("limit")); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err != nil {
			return nil, errorcode.Error(errorcode.CommonInvalidArgument)
		}
		request.Limit = int32(min(max(limit, 0), 1000))
	}
	return a.messages.ListMessages(ctx, request)
}

func (a *messageRoutes) handleSend(ctx context.Context, r *http.Request, profileID string, params map[string]string) (proto.Message, error) {
	var body sendMessageBody
	if err := readJSON(r, &body); err != nil {
		return nil, err
	}
	return a.messages.SendMessage(ctx, &messagev1.SendMessageRequest{
		ProfileId: profileID,
		PeerId:    params["peer_id"],
		Body:      body.Body,
		ClientId:  body.ClientID,
	})
}

func (a *messageRoutes) handleSendTask(ctx context.Context, r *http.Request, profileID string, params map[string]string) (proto.Message, error) {
	var body sendTaskBody
	if err := readJSON(r, &body); err != nil {
		return nil, err
	}
	request := &messagev1.SendTaskMessageRequest{
		ProfileId:   profileID,
		PeerId:      params["peer_id"],
		Title:       body.Title,
		Description: body.Description,
		Priority:    body.Priority,
		ClientId:    body.ClientID,
	}
	if raw := strings.TrimSpace(body.DueAt); raw != "" {
		due, err := time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return nil, errorcode.Error(errorcode.CommonInvalidArgument)
		}
		request.DueAt = timestamppb.New(due)
	}
	return a.messages.SendTaskMessage(ctx, request)
}

func (a *messageRoutes) handleGetMessage(ctx context.Context, _ *http.Request, profileID string, params map[string]string) (proto.Message, error) {
	return a.messages.GetMessage(ctx, &messagev1.GetMessageRequest{ProfileId: profileID, MessageId: params["message_id"]})
}

func readJSON(r *http.Request, target any) error {
	raw, err := io.ReadAll(io.LimitReader(r.Body, maxBodyBytes))
	if err != nil {
		return errorcode.Error(errorcode.CommonInvalidArgument)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return errorcode.Error(errorcode.CommonInvalidArgument)
	}
	return nil
}

func (a *messageRoutes) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
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
	case codes.InvalidArgument:
		return errorcode.CommonInvalidArgument
	case codes.NotFound:
		return errorcode.CommonNotFound
	case codes.PermissionDenied:
		return errorcode.MessageNotFriends
	case codes.Unavailable, codes.DeadlineExceeded:
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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"code":    code,
		"message": errorcode.DefaultMessage(code),
		"details": []map[string]any{{
			"@type":  "type.googleapis.com/google.rpc.ErrorInfo",
			"reason": code,
			"domain": errorcode.Domain,
		}},
	})
}
