package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"taskmanager/service/identity/internal/config"
)

var googleUserInfoURL = "https://openidconnect.googleapis.com/v1/userinfo"

var googleEndpoint = oauth2.Endpoint{
	AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
	TokenURL: "https://oauth2.googleapis.com/token",
}

type googleIdentity struct {
	Subject       string
	Email         string
	EmailVerified bool
	DisplayName   string
	AvatarURL     string
	Locale        string
}

type googleOAuth struct {
	oauth      *oauth2.Config
	httpClient *http.Client
	enabled    bool
}

func newGoogleOAuth(cfg config.Config) *googleOAuth {
	return &googleOAuth{
		oauth: &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  cfg.GoogleRedirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     googleEndpoint,
		},
		httpClient: &http.Client{Timeout: 15 * time.Second},
		enabled:    cfg.GoogleOAuthConfigured(),
	}
}

func (g *googleOAuth) authCodeURL(state string, verifier string) string {
	return g.oauth.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("prompt", "select_account"))
}

func (g *googleOAuth) exchange(ctx context.Context, code string, verifier string) (string, error) {
	token, err := g.oauth.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return "", fmt.Errorf("đổi authorization code: %w", err)
	}
	if token.AccessToken == "" {
		return "", fmt.Errorf("google không trả access token")
	}
	return token.AccessToken, nil
}

func (g *googleOAuth) userInfo(ctx context.Context, accessToken string) (googleIdentity, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, googleUserInfoURL, nil)
	if err != nil {
		return googleIdentity{}, err
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")

	response, err := g.httpClient.Do(request)
	if err != nil {
		return googleIdentity{}, fmt.Errorf("gọi google userinfo: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return googleIdentity{}, fmt.Errorf("đọc google userinfo: %w", err)
	}
	if response.StatusCode != http.StatusOK {
		return googleIdentity{}, fmt.Errorf("google userinfo trả %d: %s", response.StatusCode, string(body))
	}

	var payload struct {
		Sub           string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
		Name          string `json:"name"`
		Picture       string `json:"picture"`
		Locale        string `json:"locale"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return googleIdentity{}, fmt.Errorf("giải mã google userinfo: %w", err)
	}
	if payload.Sub == "" || payload.Email == "" {
		return googleIdentity{}, fmt.Errorf("google userinfo thiếu sub hoặc email")
	}

	return googleIdentity{
		Subject:       payload.Sub,
		Email:         payload.Email,
		EmailVerified: payload.EmailVerified,
		DisplayName:   payload.Name,
		AvatarURL:     payload.Picture,
		Locale:        payload.Locale,
	}, nil
}
