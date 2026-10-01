package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/api/idtoken"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"taskmanager/common/errorcode"
	mailv1 "taskmanager/common/gen/go/mail/v1"
	"taskmanager/service/mail-provider/internal/config"
	"taskmanager/service/mail-provider/internal/model"
)

var errUnauthorized = errors.New("pubsub notification is not authenticated")

type notificationVerifier struct {
	audience string
	account  string
}

type notificationRoutes struct {
	mail     mailv1.MailServiceClient
	verifier *notificationVerifier
	path     string
}

func newNotificationVerifier(cfg config.Config) *notificationVerifier {
	return &notificationVerifier{
		audience: cfg.PubSubAudience,
		account:  cfg.PubSubServiceAccount,
	}
}

func newNotificationRoutes(cfg config.Config, mail mailv1.MailServiceClient, verifier *notificationVerifier) *notificationRoutes {
	return &notificationRoutes{mail: mail, verifier: verifier, path: cfg.NotificationPath}
}

func newMailServiceClient(conn *grpc.ClientConn) mailv1.MailServiceClient {
	return mailv1.NewMailServiceClient(conn)
}

func (v *notificationVerifier) verify(ctx context.Context, header string) error {
	if strings.TrimSpace(v.audience) == "" {
		return nil
	}

	token := bearerToken(header)
	if token == "" {
		return errUnauthorized
	}

	payload, err := idtoken.Validate(ctx, token, v.audience)
	if err != nil {
		return fmt.Errorf("%w: %v", errUnauthorized, err)
	}
	if v.account == "" {
		return nil
	}

	email, _ := payload.Claims["email"].(string)
	verified, _ := payload.Claims["email_verified"].(bool)
	if !strings.EqualFold(strings.TrimSpace(email), strings.TrimSpace(v.account)) || !verified {
		return fmt.Errorf("%w: service account không khớp", errUnauthorized)
	}
	return nil
}

func (r *notificationRoutes) register(mux *runtime.ServeMux) error {
	return mux.HandlePath(http.MethodPost, r.path, r.handle)
}

func (r *notificationRoutes) handle(w http.ResponseWriter, req *http.Request, _ map[string]string) {
	ctx, cancel := context.WithTimeout(req.Context(), 15*time.Second)
	defer cancel()

	if err := r.verifier.verify(ctx, req.Header.Get("Authorization")); err != nil {
		slog.Warn("từ chối notice pubsub", "error", err)
		writeAPIError(w, errorcode.MailUnauthorized)
		return
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, 1<<20))
	if err != nil {
		writeAPIError(w, errorcode.MailNotificationInvalid)
		return
	}

	email, historyID, messageID, err := decodeNotice(body)
	if err != nil {
		slog.Warn("notice pubsub không hợp lệ", "error", err, "payload", envelopePayload(body))
		writeAPIError(w, errorcode.MailNotificationInvalid)
		return
	}

	response, err := r.mail.HandleNotification(ctx, &mailv1.HandleNotificationRequest{
		Email:     email,
		HistoryId: historyID,
		MessageId: messageID,
	})
	if err != nil {
		slog.Error("xử lý notice thất bại", "email", email, "history_id", historyID, "error", err)
		writeAPIError(w, grpcErrorCode(err))
		return
	}

	slog.Info("nhận notice gmail",
		"email", email,
		"history_id", historyID,
		"message_id", messageID,
		"accepted", response.GetAccepted(),
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"accepted": response.GetAccepted()})
}

func decodeNotice(body []byte) (string, string, string, error) {
	var envelope struct {
		Message struct {
			Data      string `json:"data"`
			MessageID string `json:"messageId"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return "", "", "", err
	}

	raw, err := model.DecodePubSubData(envelope.Message.Data)
	if err != nil {
		return "", "", "", err
	}

	notice, err := model.ParseGmailNotice(raw)
	if err != nil {
		return "", "", "", err
	}

	return notice.EmailAddress, notice.HistoryID, strings.TrimSpace(envelope.Message.MessageID), nil
}

func envelopePayload(body []byte) string {
	var envelope struct {
		Message struct {
			Data string `json:"data"`
		} `json:"message"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		return truncateText(string(body), 256)
	}

	raw, err := model.DecodePubSubData(envelope.Message.Data)
	if err != nil {
		return truncateText(envelope.Message.Data, 256)
	}
	return truncateText(string(raw), 256)
}

func truncateText(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
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
	case codes.NotFound:
		return errorcode.MailSubscriptionNotFound
	case codes.InvalidArgument:
		return errorcode.CommonInvalidArgument
	case codes.Unauthenticated:
		return errorcode.MailTokenRefreshFailed
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
