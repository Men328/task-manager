package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"

	"taskmanager/service/mail-provider/internal/model"
)

const pubsubScope = "https://www.googleapis.com/auth/pubsub"

const pullResponseLimit = 4 << 20

type pubsubPullClient struct {
	baseURL      string
	subscription string
	maxMessages  int
	http         *http.Client
	mu           sync.Mutex
	tokens       oauth2.TokenSource
}

func NewPubSubPullClient(baseURL string, subscription string, maxMessages int, timeout time.Duration) *pubsubPullClient {
	if maxMessages <= 0 {
		maxMessages = 10
	}
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &pubsubPullClient{
		baseURL:      strings.TrimRight(baseURL, "/"),
		subscription: strings.TrimSpace(subscription),
		maxMessages:  maxMessages,
		http:         &http.Client{Timeout: timeout},
	}
}

func (c *pubsubPullClient) Receive(ctx context.Context) ([]model.PulledNotice, error) {
	body, err := c.call(ctx, "pull", map[string]any{
		"maxMessages":       c.maxMessages,
		"returnImmediately": false,
	})
	if err != nil {
		return nil, err
	}

	var response struct {
		ReceivedMessages []struct {
			AckID   string `json:"ackId"`
			Message struct {
				Data      string `json:"data"`
				MessageID string `json:"messageId"`
			} `json:"message"`
		} `json:"receivedMessages"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("giải mã pull response: %w", err)
	}

	out := make([]model.PulledNotice, 0, len(response.ReceivedMessages))
	for _, received := range response.ReceivedMessages {
		item := model.PulledNotice{AckID: strings.TrimSpace(received.AckID)}
		raw, decodeErr := model.DecodePubSubData(received.Message.Data)
		if decodeErr != nil {
			item.Raw = "message.data không decode được base64"
			out = append(out, item)
			continue
		}
		notice, parseErr := model.ParseGmailNotice(raw)
		if parseErr != nil {
			item.Raw = truncate(string(raw), 256)
			out = append(out, item)
			continue
		}
		notice.MessageID = strings.TrimSpace(received.Message.MessageID)
		item.Notice = notice
		out = append(out, item)
	}
	return out, nil
}

func (c *pubsubPullClient) Acknowledge(ctx context.Context, ackIDs []string) error {
	if len(ackIDs) == 0 {
		return nil
	}
	_, err := c.call(ctx, "acknowledge", map[string]any{"ackIds": ackIDs})
	return err
}

func (c *pubsubPullClient) call(ctx context.Context, method string, payload any) ([]byte, error) {
	if c.subscription == "" {
		return nil, model.ErrNotConfigured
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("mã hoá %s request: %w", method, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/"+c.subscription+":"+method, bytes.NewReader(encoded))
	if err != nil {
		return nil, fmt.Errorf("tạo %s request: %w", method, err)
	}

	token, err := c.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("gọi pubsub %s: %w", method, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, pullResponseLimit))
	if err != nil {
		return nil, fmt.Errorf("đọc %s response: %w", method, err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pubsub %s trả %d: %s", method, response.StatusCode, truncate(string(body), 512))
	}
	return body, nil
}

func (c *pubsubPullClient) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	source := c.tokens
	if source == nil {
		created, err := google.DefaultTokenSource(ctx, pubsubScope)
		if err != nil {
			c.mu.Unlock()
			return "", fmt.Errorf("không tạo được token source cho pubsub: %w", err)
		}
		c.tokens = created
		source = created
	}
	c.mu.Unlock()

	token, err := source.Token()
	if err != nil {
		return "", fmt.Errorf("lấy access token pubsub: %w", err)
	}
	return token.AccessToken, nil
}
