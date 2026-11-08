package repository

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"taskmanager/service/message/internal/model"
)

const MessageEventName = "message.created"

type SoketiBroadcaster struct {
	baseURL       string
	appID         string
	appKey        string
	appSecret     string
	channelPrefix string
	client        *http.Client
	now           func() time.Time
}

type soketiMessage struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversationId"`
	SenderID       string `json:"senderId"`
	RecipientID    string `json:"recipientId"`
	Body           string `json:"body"`
	ClientID       string `json:"clientId"`
	CreatedAt      string `json:"createdAt"`
}

type soketiEvent struct {
	Name     string   `json:"name"`
	Channels []string `json:"channels"`
	Data     string   `json:"data"`
}

func NewSoketiBroadcaster(baseURL, appID, appKey, appSecret, channelPrefix string, timeout time.Duration) *SoketiBroadcaster {
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	return &SoketiBroadcaster{
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		appID:         strings.TrimSpace(appID),
		appKey:        strings.TrimSpace(appKey),
		appSecret:     strings.TrimSpace(appSecret),
		channelPrefix: channelPrefix,
		client:        &http.Client{Timeout: timeout},
		now:           time.Now,
	}
}

func ChatChannelName(prefix, profileID string) string {
	return "private-" + prefix + profileID
}

func (b *SoketiBroadcaster) Broadcast(ctx context.Context, message model.Message) error {
	payload, err := json.Marshal(soketiMessage{
		ID:             message.ID,
		ConversationID: message.ConversationID,
		SenderID:       message.SenderID,
		RecipientID:    message.RecipientID,
		Body:           message.Body,
		ClientID:       message.ClientID,
		CreatedAt:      message.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("mã hoá tin nhắn: %w", err)
	}

	event, err := json.Marshal(soketiEvent{
		Name: MessageEventName,
		Channels: []string{
			ChatChannelName(b.channelPrefix, message.RecipientID),
			ChatChannelName(b.channelPrefix, message.SenderID),
		},
		Data: string(payload),
	})
	if err != nil {
		return fmt.Errorf("mã hoá sự kiện soketi: %w", err)
	}

	path := "/apps/" + b.appID + "/events"
	endpoint := b.baseURL + path + "?" + b.authQuery(http.MethodPost, path, event)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(event))
	if err != nil {
		return fmt.Errorf("tạo request soketi: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := b.client.Do(request)
	if err != nil {
		return fmt.Errorf("gọi soketi: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 512))
		return fmt.Errorf("soketi trả HTTP %d: %s", response.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (b *SoketiBroadcaster) authQuery(method, path string, body []byte) string {
	sum := md5.Sum(body)

	params := url.Values{}
	params.Set("auth_key", b.appKey)
	params.Set("auth_timestamp", strconv.FormatInt(b.now().Unix(), 10))
	params.Set("auth_version", "1.0")
	params.Set("body_md5", hex.EncodeToString(sum[:]))

	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params.Get(key))
	}

	mac := hmac.New(sha256.New, []byte(b.appSecret))
	mac.Write([]byte(method + "\n" + path + "\n" + strings.Join(parts, "&")))
	params.Set("auth_signature", hex.EncodeToString(mac.Sum(nil)))

	return params.Encode()
}

type NoopBroadcaster struct{}

func (NoopBroadcaster) Broadcast(context.Context, model.Message) error { return nil }
