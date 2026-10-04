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

	"taskmanager/common/errorcode"
	notificationv1 "taskmanager/common/gen/go/notification/v1"
)

const routeTimeout = 10 * time.Second

type noticeRoutes struct {
	notices notificationv1.NoticeServiceClient
	signer  *sessionSigner
}

type markReadBody struct {
	Id  string   `json:"id"`
	Ids []string `json:"ids"`
}

func newNoticeRoutes(notices notificationv1.NoticeServiceClient, signer *sessionSigner) *noticeRoutes {
	return &noticeRoutes{notices: notices, signer: signer}
}

func (a *noticeRoutes) register(mux *runtime.ServeMux) error {
	routes := []struct {
		method  string
		path    string
		handler runtime.HandlerFunc
	}{
		{method: http.MethodGet, path: "/v1/notices", handler: a.handleList},
		{method: http.MethodGet, path: "/v1/notices/unread-count", handler: a.handleCount},
		{method: http.MethodPost, path: "/v1/notices/read", handler: a.handleMarkRead},
	}

	for _, route := range routes {
		if err := mux.HandlePath(route.method, route.path, route.handler); err != nil {
			return err
		}
	}
	return nil
}

func (a *noticeRoutes) handleList(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	profileID, ok := a.authenticate(w, r)
	if !ok {
		return
	}

	request := &notificationv1.ListNoticesRequest{ProfileId: profileID}
	if limit := strings.TrimSpace(r.URL.Query().Get("limit")); limit != "" {
		if parsed, err := strconv.Atoi(limit); err == nil && parsed > 0 {
			request.Limit = int32(parsed)
		}
	}
	request.UnreadOnly = parseBool(r.URL.Query().Get("unread_only"))

	ctx, cancel := context.WithTimeout(r.Context(), routeTimeout)
	defer cancel()

	response, err := a.notices.ListNotices(ctx, request)
	if err != nil {
		writeAPIError(w, grpcErrorCode(err))
		return
	}
	writeProto(w, response)
}

func (a *noticeRoutes) handleCount(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	profileID, ok := a.authenticate(w, r)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), routeTimeout)
	defer cancel()

	response, err := a.notices.CountUnreadNotices(ctx, &notificationv1.CountUnreadNoticesRequest{ProfileId: profileID})
	if err != nil {
		writeAPIError(w, grpcErrorCode(err))
		return
	}
	writeProto(w, response)
}

func (a *noticeRoutes) handleMarkRead(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	profileID, ok := a.authenticate(w, r)
	if !ok {
		return
	}

	raw, err := io.ReadAll(io.LimitReader(r.Body, 64*1024))
	if err != nil {
		writeAPIError(w, errorcode.CommonInvalidArgument)
		return
	}

	var body markReadBody
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &body); err != nil {
			writeAPIError(w, errorcode.CommonInvalidArgument)
			return
		}
	}

	ids := body.Ids
	if strings.TrimSpace(body.Id) != "" {
		ids = append(ids, body.Id)
	}

	ctx, cancel := context.WithTimeout(r.Context(), routeTimeout)
	defer cancel()

	response, err := a.notices.MarkNoticesRead(ctx, &notificationv1.MarkNoticesReadRequest{
		ProfileId: profileID,
		Ids:       ids,
	})
	if err != nil {
		writeAPIError(w, grpcErrorCode(err))
		return
	}
	writeProto(w, response)
}

func (a *noticeRoutes) authenticate(w http.ResponseWriter, r *http.Request) (string, bool) {
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

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
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
		return errorcode.NotificationNotFound
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
