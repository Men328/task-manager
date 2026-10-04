package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"taskmanager/service/notification/internal/config"
)

const sessionIssuer = "taskmanager-identity"

var errSessionNotConfigured = errors.New("session secret chưa được cấu hình")

type sessionClaims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

type sessionSigner struct {
	secret []byte
}

func newSessionSigner(cfg config.Config) *sessionSigner {
	return &sessionSigner{secret: []byte(cfg.SessionSecret)}
}

func (s *sessionSigner) configured() bool {
	return len(s.secret) >= 16
}

func (s *sessionSigner) verify(raw string) (string, error) {
	if !s.configured() {
		return "", errSessionNotConfigured
	}

	var claims sessionClaims
	_, err := jwt.ParseWithClaims(raw, &claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("thuật toán ký không hợp lệ: %s", token.Method.Alg())
		}
		return s.secret, nil
	}, jwt.WithIssuer(sessionIssuer), jwt.WithExpirationRequired())
	if err != nil {
		return "", err
	}

	subject := strings.TrimSpace(claims.Subject)
	if subject == "" {
		return "", errors.New("session token thiếu subject")
	}
	return subject, nil
}
