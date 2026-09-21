package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	"taskmanager/common/errorcode"
	identityv1 "taskmanager/common/gen/go/identity/v1"
	"taskmanager/service/identity/internal/config"
)

const oauthStateTTL = 10 * time.Minute

type authRoutes struct {
	profiles    identityv1.ProfileServiceClient
	google      *googleOAuth
	signer      *sessionSigner
	frontendURL string
}

func newAuthRoutes(cfg config.Config, profiles identityv1.ProfileServiceClient, google *googleOAuth, signer *sessionSigner) *authRoutes {
	return &authRoutes{
		profiles:    profiles,
		google:      google,
		signer:      signer,
		frontendURL: cfg.FrontendBaseURL,
	}
}

func newProfileServiceClient(conn *grpc.ClientConn) identityv1.ProfileServiceClient {
	return identityv1.NewProfileServiceClient(conn)
}

func (a *authRoutes) register(mux *runtime.ServeMux) error {
	routes := []struct {
		path    string
		handler runtime.HandlerFunc
	}{
		{path: "/v1/auth/google/login", handler: a.handleGoogleLogin},
		{path: "/v1/auth/google/callback", handler: a.handleGoogleCallback},
		{path: "/v1/auth/me", handler: a.handleMe},
	}

	for _, route := range routes {
		if err := mux.HandlePath(http.MethodGet, route.path, route.handler); err != nil {
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
		ExpiresAt: time.Now().Add(oauthStateTTL).Unix(),
	})
	if err != nil {
		slog.Error("ký oauth state", "error", err)
		a.redirectError(w, r, errorcode.IdentityAuthNotConfigured)
		return
	}

	http.Redirect(w, r, a.google.authCodeURL(state, verifier), http.StatusFound)
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

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	accessToken, err := a.google.exchange(ctx, code, state.Verifier)
	if err != nil {
		slog.Error("đổi google authorization code", "error", err)
		a.redirectError(w, r, errorcode.IdentityAuthExchangeFailed)
		return
	}

	identity, err := a.google.userInfo(ctx, accessToken)
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
	token, expiresAt, err := a.signer.issue(profile.GetId(), profile.GetEmail(), time.Now())
	if err != nil {
		slog.Error("phát session token", "error", err)
		a.redirectError(w, r, errorcode.IdentityAuthNotConfigured)
		return
	}

	slog.Info("đăng nhập google thành công",
		"profile_id", profile.GetId(),
		"created", response.GetCreated(),
		"expires_at", expiresAt,
	)
	http.Redirect(w, r, a.callbackURL("token", token), http.StatusFound)
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

func (a *authRoutes) callbackURL(key string, value string) string {
	values := url.Values{}
	values.Set(key, value)
	return a.frontendURL + "/auth/callback#" + values.Encode()
}

func (a *authRoutes) redirectError(w http.ResponseWriter, r *http.Request, code string) {
	http.Redirect(w, r, a.callbackURL("error", code), http.StatusFound)
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
	message, _ := errorcode.Message(code, "vi")

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
