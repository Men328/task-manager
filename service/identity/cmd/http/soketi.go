package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"taskmanager/common/errorcode"
	"taskmanager/service/identity/internal/config"
)

const (
	defaultChannelPrefix     = "noti-internal-"
	defaultChatChannelPrefix = "chat-"
)

type soketiAuth struct {
	appKey    string
	appSecret string
	prefixes  []string
	signer    *sessionSigner
}

type soketiAuthRequest struct {
	SocketID    string `json:"socket_id"`
	ChannelName string `json:"channel_name"`
	Token       string `json:"token"`
}

func newSoketiAuth(cfg config.Config, signer *sessionSigner) *soketiAuth {
	return &soketiAuth{
		appKey:    strings.TrimSpace(cfg.SoketiAppKey),
		appSecret: strings.TrimSpace(cfg.SoketiAppSecret),
		prefixes: []string{
			prefixOr(cfg.SoketiChannelPrefix, defaultChannelPrefix),
			prefixOr(cfg.SoketiChatChannelPrefix, defaultChatChannelPrefix),
		},
		signer: signer,
	}
}

func prefixOr(value, fallback string) string {
	if prefix := strings.TrimSpace(value); prefix != "" {
		return prefix
	}
	return fallback
}

func (s *soketiAuth) configured() bool {
	return s.appKey != "" && s.appSecret != ""
}

func (s *soketiAuth) ownsChannel(profileID, channelName string) bool {
	for _, prefix := range s.prefixes {
		if channelName == "private-"+prefix+profileID {
			return true
		}
	}
	return false
}

func (s *soketiAuth) handle(w http.ResponseWriter, r *http.Request, _ map[string]string) {
	if !s.configured() {
		writeAPIError(w, errorcode.IdentityAuthSoketiNotConfigured)
		return
	}

	request, err := s.parse(r)
	if err != nil {
		writeAPIError(w, errorcode.IdentityAuthSoketiRequestInvalid)
		return
	}
	if strings.TrimSpace(request.SocketID) == "" || strings.TrimSpace(request.ChannelName) == "" {
		writeAPIError(w, errorcode.IdentityAuthSoketiRequestInvalid)
		return
	}

	token := strings.TrimSpace(request.Token)
	if token == "" {
		token = bearerToken(r.Header.Get("Authorization"))
	}
	if token == "" {
		writeAPIError(w, errorcode.IdentityAuthTokenMissing)
		return
	}

	claims, err := s.signer.verify(token)
	if err != nil {
		writeAPIError(w, errorcode.IdentityAuthTokenInvalid)
		return
	}

	if !s.ownsChannel(claims.Subject, request.ChannelName) {
		writeAPIError(w, errorcode.IdentityAuthSoketiChannelForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"auth": s.appKey + ":" + s.sign(request.SocketID, request.ChannelName),
	})
}

func (s *soketiAuth) parse(r *http.Request) (soketiAuthRequest, error) {
	if strings.Contains(r.Header.Get("Content-Type"), "application/json") {
		raw, err := io.ReadAll(io.LimitReader(r.Body, 16*1024))
		if err != nil {
			return soketiAuthRequest{}, err
		}
		var request soketiAuthRequest
		if err := json.Unmarshal(raw, &request); err != nil {
			return soketiAuthRequest{}, err
		}
		return request, nil
	}

	if err := r.ParseForm(); err != nil {
		return soketiAuthRequest{}, err
	}
	return soketiAuthRequest{
		SocketID:    r.Form.Get("socket_id"),
		ChannelName: r.Form.Get("channel_name"),
		Token:       r.Form.Get("token"),
	}, nil
}

func (s *soketiAuth) sign(socketID string, channelName string) string {
	mac := hmac.New(sha256.New, []byte(s.appSecret))
	mac.Write([]byte(socketID + ":" + channelName))
	return hex.EncodeToString(mac.Sum(nil))
}
