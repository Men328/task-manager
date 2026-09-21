package main

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"taskmanager/service/identity/internal/config"
)

const sessionIssuer = "taskmanager-identity"

var (
	errSessionNotConfigured = errors.New("session secret chưa được cấu hình")
	errStateInvalid         = errors.New("state không hợp lệ hoặc đã hết hạn")
)

type sessionClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type sessionSigner struct {
	secret []byte
	ttl    time.Duration
}

func newSessionSigner(cfg config.Config) *sessionSigner {
	return &sessionSigner{secret: []byte(cfg.SessionSecret), ttl: cfg.SessionTTL}
}

func (s *sessionSigner) configured() bool {
	return len(s.secret) >= 16
}

func (s *sessionSigner) issue(profileID string, email string, now time.Time) (string, time.Time, error) {
	if !s.configured() {
		return "", time.Time{}, errSessionNotConfigured
	}

	expiresAt := now.Add(s.ttl)
	claims := sessionClaims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   profileID,
			Issuer:    sessionIssuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("ký session token: %w", err)
	}
	return signed, expiresAt, nil
}

func (s *sessionSigner) verify(raw string) (sessionClaims, error) {
	if !s.configured() {
		return sessionClaims{}, errSessionNotConfigured
	}

	var claims sessionClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("thuật toán ký không hợp lệ: %s", token.Method.Alg())
		}
		return s.secret, nil
	}, jwt.WithIssuer(sessionIssuer), jwt.WithExpirationRequired())
	if err != nil {
		return sessionClaims{}, err
	}
	if strings.TrimSpace(claims.Subject) == "" {
		return sessionClaims{}, errors.New("session token thiếu subject")
	}
	return claims, nil
}

type oauthState struct {
	Nonce     string `json:"n"`
	Verifier  string `json:"v"`
	ExpiresAt int64  `json:"e"`
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *sessionSigner) signState(state oauthState) (string, error) {
	if !s.configured() {
		return "", errSessionNotConfigured
	}

	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + s.stateMAC(encoded), nil
}

func (s *sessionSigner) verifyState(raw string) (oauthState, error) {
	if !s.configured() {
		return oauthState{}, errSessionNotConfigured
	}

	encoded, mac, found := strings.Cut(raw, ".")
	if !found || encoded == "" || mac == "" {
		return oauthState{}, errStateInvalid
	}
	if !hmac.Equal([]byte(mac), []byte(s.stateMAC(encoded))) {
		return oauthState{}, errStateInvalid
	}

	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return oauthState{}, errStateInvalid
	}

	var state oauthState
	if err := json.Unmarshal(payload, &state); err != nil {
		return oauthState{}, errStateInvalid
	}
	if state.ExpiresAt <= time.Now().Unix() {
		return oauthState{}, errStateInvalid
	}
	if state.Verifier == "" {
		return oauthState{}, errStateInvalid
	}
	return state, nil
}

func (s *sessionSigner) stateMAC(encoded string) string {
	mac := hmac.New(sha256.New, s.secret)
	mac.Write([]byte(encoded))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
