package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type tokenRefresher struct {
	clientID     string
	clientSecret string
	tokenURL     string
	http         *http.Client
}

func NewTokenRefresher(clientID string, clientSecret string, tokenURL string, timeout time.Duration) *tokenRefresher {
	return &tokenRefresher{
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     tokenURL,
		http:         &http.Client{Timeout: timeout},
	}
}

func (r *tokenRefresher) Refresh(ctx context.Context, refreshToken string) (model.Token, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", r.clientID)
	form.Set("client_secret", r.clientSecret)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return model.Token{}, fmt.Errorf("%w: tạo request: %v", model.ErrTokenRefresh, err)
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("Accept", "application/json")

	response, err := r.http.Do(request)
	if err != nil {
		return model.Token{}, fmt.Errorf("%w: %v", model.ErrTokenRefresh, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return model.Token{}, fmt.Errorf("%w: đọc response: %v", model.ErrTokenRefresh, err)
	}

	var payload struct {
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
		ExpiresIn        int64  `json:"expires_in"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return model.Token{}, fmt.Errorf("%w: giải mã response: %v", model.ErrTokenRefresh, err)
	}
	if response.StatusCode != http.StatusOK || payload.Error != "" {
		return model.Token{}, fmt.Errorf("%w: %s %s", model.ErrTokenRefresh, payload.Error, payload.ErrorDescription)
	}
	if payload.AccessToken == "" {
		return model.Token{}, fmt.Errorf("%w: google không trả access token", model.ErrTokenRefresh)
	}

	expiresAt := time.Now().UTC().Add(time.Hour)
	if payload.ExpiresIn > 0 {
		expiresAt = time.Now().UTC().Add(time.Duration(payload.ExpiresIn) * time.Second)
	}

	return model.Token{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}
