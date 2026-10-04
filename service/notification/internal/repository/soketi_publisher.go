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

	"taskmanager/service/notification/internal/model"
)

const soketiEventName = "notice.created"

type SoketiPublisher struct {
	baseURL       string
	appID         string
	appKey        string
	appSecret     string
	channelPrefix string
	client        *http.Client
}

type soketiNotice struct {
	ID         string `json:"id"`
	ProfileID  string `json:"profileId"`
	Type       string `json:"type"`
	Title      string `json:"title"`
	Body       string `json:"body"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Source     string `json:"source"`
	IsRead     bool   `json:"isRead"`
	CreatedAt  string `json:"createdAt"`
}

type soketiEvent struct {
	Name     string   `json:"name"`
	Channels []string `json:"channels"`
	Data     string   `json:"data"`
}

func NewSoketiPublisher(baseURL string, appID string, appKey string, appSecret string, channelPrefix string, timeout time.Duration) *SoketiPublisher {
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	return &SoketiPublisher{
		baseURL:       strings.TrimRight(strings.TrimSpace(baseURL), "/"),
		appID:         strings.TrimSpace(appID),
		appKey:        strings.TrimSpace(appKey),
		appSecret:     strings.TrimSpace(appSecret),
		channelPrefix: channelPrefix,
		client:        &http.Client{Timeout: timeout},
	}
}

func (p *SoketiPublisher) Configured() bool {
	return p.baseURL != "" && p.appID != "" && p.appKey != "" && p.appSecret != ""
}

func (p *SoketiPublisher) Publish(ctx context.Context, notice model.Notice) error {
	if !p.Configured() {
		return fmt.Errorf("soketi chưa được cấu hình")
	}

	payload, err := json.Marshal(soketiNotice{
		ID:         notice.ID,
		ProfileID:  notice.ProfileID,
		Type:       notice.Type,
		Title:      notice.Title,
		Body:       notice.Body,
		TargetType: notice.TargetType,
		TargetID:   notice.TargetID,
		Source:     notice.Source,
		IsRead:     notice.IsRead,
		CreatedAt:  notice.CreatedAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("mã hoá notice: %w", err)
	}

	event, err := json.Marshal(soketiEvent{
		Name:     soketiEventName,
		Channels: []string{model.ChannelName(p.channelPrefix, notice.ProfileID)},
		Data:     string(payload),
	})
	if err != nil {
		return fmt.Errorf("mã hoá sự kiện soketi: %w", err)
	}

	path := "/apps/" + p.appID + "/events"
	endpoint := p.baseURL + path + "?" + p.authQuery(http.MethodPost, path, event)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(event))
	if err != nil {
		return fmt.Errorf("tạo request soketi: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := p.client.Do(request)
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

func (p *SoketiPublisher) authQuery(method string, path string, body []byte) string {
	sum := md5.Sum(body)

	params := url.Values{}
	params.Set("auth_key", p.appKey)
	params.Set("auth_timestamp", strconv.FormatInt(time.Now().Unix(), 10))
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
	query := strings.Join(parts, "&")

	mac := hmac.New(sha256.New, []byte(p.appSecret))
	mac.Write([]byte(method + "\n" + path + "\n" + query))
	params.Set("auth_signature", hex.EncodeToString(mac.Sum(nil)))

	return params.Encode()
}
