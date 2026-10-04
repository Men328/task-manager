package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"golang.org/x/oauth2"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	identityv1 "taskmanager/common/gen/go/identity/v1"
	mailv1 "taskmanager/common/gen/go/mail/v1"
	"taskmanager/service/identity/internal/config"
)

const oauthStateTTL = 10 * time.Minute

type authRoutes struct {
	profiles    identityv1.ProfileServiceClient
	mail        mailv1.MailServiceClient
	google      *googleOAuth
	signer      *sessionSigner
	soketi      *soketiAuth
	frontendURL string
}

func newAuthRoutes(
	cfg config.Config,
	profiles identityv1.ProfileServiceClient,
	mail mailv1.MailServiceClient,
	google *googleOAuth,
	signer *sessionSigner,
	soketi *soketiAuth,
) *authRoutes {
	return &authRoutes{
		profiles:    profiles,
		mail:        mail,
		google:      google,
		signer:      signer,
		soketi:      soketi,
		frontendURL: cfg.FrontendBaseURL,
	}
}

func newProfileServiceClient(conn *grpc.ClientConn) identityv1.ProfileServiceClient {
	return identityv1.NewProfileServiceClient(conn)
}

func newMailServiceClient(conn *grpc.ClientConn) mailv1.MailServiceClient {
	return mailv1.NewMailServiceClient(conn)
}

func (a *authRoutes) register(mux *runtime.ServeMux) error {
	routes := []struct {
		method  string
		path    string
		handler runtime.HandlerFunc
	}{
		{method: http.MethodGet, path: "/v1/auth/google/login", handler: a.handleGoogleLogin},
		{method: http.MethodGet, path: "/v1/auth/google/callback", handler: a.handleGoogleCallback},
		{method: http.MethodGet, path: "/v1/auth/me", handler: a.handleMe},
	}

	for _, route := range routes {
		if err := mux.HandlePath(route.method, route.path, route.handler); err != nil {
			return err
		}
	}

	if a.soketi != nil {
		if err := mux.HandlePath(http.MethodPost, "/v1/auth/soketi", a.soketi.handle); err != nil {
			return err
		}
	}
	return nil
}

func (a *authRoutes) handleGoogleLogin(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	if !a.google.enabled {
		a.redirectError(w, r, errorcode.IdentityAuthNotConfigured)
		return
	}

	gmail := wantsGmail(r.URL.Query().Get("gmail"))

	nonce, err := randomToken(24)
	if err != nil {
		a.redirectError(w, r, errorcode.CommonInternal)
		return
	}
	verifier, err := randomToken(48)
	if err != nil {
		a.redirectError(w, r, errorcode.CommonInternal)
		return
	}

	state, err := a.signer.signState(oauthState{
		Nonce:     nonce,
		Verifier:  verifier,
		Gmail:     gmail,
		ExpiresAt: time.Now().Add(oauthStateTTL).Unix(),
	})
	if err != nil {
		slog.Error("ký oauth state", "error", err)
		a.redirectError(w, r, errorcode.IdentityAuthNotConfigured)
		return
	}

	http.Redirect(w, r, a.google.authCodeURL(state, verifier, gmail), http.StatusFound)
}

func (a *authRoutes) handleGoogleCallback(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	query := r.URL.Query()

	if providerError := strings.TrimSpace(query.Get("error")); providerError != "" {
		slog.Warn("google trả lỗi uỷ quyền", "error", providerError, "description", query.Get("error_description"))
		a.redirectError(w, r, errorcode.IdentityAuthExchangeFailed)
		return
	}

	code := strings.TrimSpace(query.Get("code"))
	if code == "" {
		a.redirectError(w, r, errorcode.IdentityAuthCodeRequired)
		return
	}

	state, err := a.signer.verifyState(query.Get("state"))
	if err != nil {
		a.redirectError(w, r, errorcode.IdentityAuthStateInvalid)
		return
	}

	if !a.google.enabled {
		a.redirectError(w, r, errorcode.IdentityAuthNotConfigured)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	token, err := a.google.exchange(ctx, code, state.Verifier)
	if err != nil {
		slog.Error("đổi google authorization code", "error", err)
		a.redirectError(w, r, errorcode.IdentityAuthExchangeFailed)
		return
	}

	identity, err := a.google.userInfo(ctx, token.AccessToken)
	if err != nil {
		slog.Error("lấy google userinfo", "error", err)
		a.redirectError(w, r, errorcode.IdentityAuthExchangeFailed)
		return
	}
	if !identity.EmailVerified {
		slog.Warn("google trả email chưa xác minh", "email", identity.Email)
		a.redirectError(w, r, errorcode.IdentityAuthExchangeFailed)
		return
	}

	response, err := a.profiles.LoginWithProvider(ctx, &identityv1.LoginWithProviderRequest{
		Provider:       "google",
		ProviderUserId: identity.Subject,
		Email:          identity.Email,
		DisplayName:    identity.DisplayName,
		AvatarUrl:      identity.AvatarURL,
		Locale:         identity.Locale,
	})
	if err != nil {
		slog.Error("upsert profile từ google", "error", err, "email", identity.Email)
		a.redirectError(w, r, grpcErrorCode(err))
		return
	}

	profile := response.GetProfile()
	sessionToken, expiresAt, err := a.signer.issue(profile.GetId(), profile.GetEmail(), time.Now())
	if err != nil {
		slog.Error("phát session token", "error", err)
		a.redirectError(w, r, errorcode.IdentityAuthNotConfigured)
		return
	}

	values := url.Values{}
	values.Set("token", sessionToken)

	if state.Gmail {
		if subscribeErr := a.subscribeMailbox(ctx, profile.GetId(), profile.GetEmail(), token); subscribeErr != nil {
			slog.Error("đăng ký nhận thông báo gmail thất bại",
				"profile_id", profile.GetId(),
				"email", profile.GetEmail(),
				"error", subscribeErr,
			)
			values.Set("mail", "error")
		} else {
			values.Set("mail", "ok")
		}
	}

	slog.Info("đăng nhập google thành công",
		"profile_id", profile.GetId(),
		"created", response.GetCreated(),
		"expires_at", expiresAt,
		"gmail", state.Gmail,
	)
	http.Redirect(w, r, a.callbackURL(values), http.StatusFound)
}

func (a *authRoutes) subscribeMailbox(ctx context.Context, profileID string, email string, token *oauth2.Token) error {
	if a.mail == nil {
		return fmt.Errorf("mail service client chưa được cấu hình")
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		slog.Warn("google không trả refresh token; watch sẽ hết hạn sau khoảng 1 giờ", "email", email)
	}

	expiresAt := token.Expiry
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(time.Hour)
	}

	_, err := a.mail.Subscribe(ctx, &mailv1.SubscribeRequest{
		ProfileId:            profileID,
		Email:                email,
		AccessToken:          token.AccessToken,
		RefreshToken:         token.RefreshToken,
		AccessTokenExpiresAt: timestamppb.New(expiresAt),
	})
	return err
}

func (a *authRoutes) handleMe(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	raw := bearerToken(r.Header.Get("Authorization"))
	if raw == "" {
		writeAPIError(w, errorcode.IdentityAuthTokenMissing)
		return
	}

	claims, err := a.signer.verify(raw)
	if err != nil {
		writeAPIError(w, errorcode.IdentityAuthTokenInvalid)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response, err := a.profiles.GetProfile(ctx, &identityv1.GetProfileRequest{Id: claims.Subject})
	if err != nil {
		writeAPIError(w, grpcErrorCode(err))
		return
	}

	encoded, err := protojson.MarshalOptions{EmitUnpopulated: true}.Marshal(response.GetProfile())
	if err != nil {
		writeAPIError(w, errorcode.CommonInternal)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(struct {
		Profile json.RawMessage `json:"profile"`
	}{Profile: encoded})
}

func (a *authRoutes) callbackURL(values url.Values) string {
	return a.frontendURL + "/auth/callback#" + values.Encode()
}

func (a *authRoutes) redirectError(w http.ResponseWriter, r *http.Request, code string) {
	values := url.Values{}
	values.Set("error", code)
	http.Redirect(w, r, a.callbackURL(values), http.StatusFound)
}

func wantsGmail(value string) bool {
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
		return errorcode.IdentityProfileNotFound
	case codes.InvalidArgument:
		return errorcode.CommonInvalidArgument
	case codes.AlreadyExists:
		return errorcode.CommonAlreadyExists
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
