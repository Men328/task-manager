package repository

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

const gmailResponseLimit = 4 << 20

const gmailAttachmentLimit = 32 << 20

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

var whitespacePattern = regexp.MustCompile(`\s+`)

type gmailClient struct {
	baseURL      string
	http         *http.Client
	maxBodyBytes int
}

type gmailPayload struct {
	MimeType string `json:"mimeType"`
	Filename string `json:"filename"`
	Headers  []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	} `json:"headers"`
	Body struct {
		Data         string `json:"data"`
		AttachmentID string `json:"attachmentId"`
		Size         int64  `json:"size"`
	} `json:"body"`
	Parts []gmailPayload `json:"parts"`
}

type gmailMessage struct {
	ID           string        `json:"id"`
	ThreadID     string        `json:"threadId"`
	Snippet      string        `json:"snippet"`
	InternalDate string        `json:"internalDate"`
	Payload      *gmailPayload `json:"payload"`
}

func NewGmailClient(baseURL string, timeout time.Duration, maxBodyBytes int) *gmailClient {
	if maxBodyBytes <= 0 {
		maxBodyBytes = model.DefaultMaxBodyBytes
	}
	return &gmailClient{
		baseURL:      strings.TrimRight(baseURL, "/"),
		http:         &http.Client{Timeout: timeout},
		maxBodyBytes: maxBodyBytes,
	}
}

func (c *gmailClient) Watch(ctx context.Context, accessToken string, topicName string, labelIDs []string) (model.WatchResult, error) {
	payload := map[string]any{"topicName": topicName}
	if len(labelIDs) > 0 {
		payload["labelIds"] = labelIDs
		payload["labelFilterBehavior"] = "INCLUDE"
	}

	body, status, err := c.request(ctx, http.MethodPost, "/gmail/v1/users/me/watch", accessToken, payload)
	if err != nil {
		return model.WatchResult{}, err
	}
	if status != http.StatusOK {
		return model.WatchResult{}, gmailStatusError(status, body)
	}

	var raw struct {
		HistoryID  string `json:"historyId"`
		Expiration string `json:"expiration"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return model.WatchResult{}, fmt.Errorf("%w: giải mã watch: %v", model.ErrGmailFailed, err)
	}

	result := model.WatchResult{HistoryID: raw.HistoryID}
	if millis, parseErr := strconv.ParseInt(raw.Expiration, 10, 64); parseErr == nil && millis > 0 {
		result.ExpiresAt = time.UnixMilli(millis).UTC()
	}
	return result, nil
}

func (c *gmailClient) Stop(ctx context.Context, accessToken string) error {
	body, status, err := c.request(ctx, http.MethodPost, "/gmail/v1/users/me/stop", accessToken, nil)
	if err != nil {
		return err
	}
	if status != http.StatusOK && status != http.StatusNoContent {
		return gmailStatusError(status, body)
	}
	return nil
}

func (c *gmailClient) NewMessageIDs(ctx context.Context, accessToken string, startHistoryID string, labelIDs []string) ([]string, string, error) {
	if strings.TrimSpace(startHistoryID) == "" {
		current, err := c.ProfileHistoryID(ctx, accessToken)
		if err != nil {
			return nil, "", err
		}
		return nil, current, nil
	}

	values := url.Values{}
	values.Set("startHistoryId", startHistoryID)
	values.Add("historyTypes", "messageAdded")
	for _, label := range labelIDs {
		if trimmed := strings.TrimSpace(label); trimmed != "" {
			values.Add("labelId", trimmed)
		}
	}

	body, status, err := c.request(ctx, http.MethodGet, "/gmail/v1/users/me/history?"+values.Encode(), accessToken, nil)
	if err != nil {
		return nil, "", err
	}
	if status == http.StatusNotFound {
		return nil, "", model.ErrHistoryGone
	}
	if status != http.StatusOK {
		return nil, "", gmailStatusError(status, body)
	}

	var raw struct {
		History []struct {
			Messages []struct {
				ID string `json:"id"`
			} `json:"messages"`
		} `json:"history"`
		HistoryID string `json:"historyId"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, "", fmt.Errorf("%w: giải mã history: %v", model.ErrGmailFailed, err)
	}

	seen := map[string]bool{}
	ids := make([]string, 0, len(raw.History))
	for _, entry := range raw.History {
		for _, message := range entry.Messages {
			if message.ID == "" || seen[message.ID] {
				continue
			}
			seen[message.ID] = true
			ids = append(ids, message.ID)
		}
	}
	return ids, raw.HistoryID, nil
}

func (c *gmailClient) Message(ctx context.Context, accessToken string, messageID string) (model.EmailMessage, error) {
	path := "/gmail/v1/users/me/messages/" + url.PathEscape(messageID) + "?format=full"
	body, status, err := c.request(ctx, http.MethodGet, path, accessToken, nil)
	if err != nil {
		return model.EmailMessage{}, err
	}
	if status != http.StatusOK {
		return model.EmailMessage{}, gmailStatusError(status, body)
	}

	var raw gmailMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return model.EmailMessage{}, fmt.Errorf("%w: giải mã message: %v", model.ErrGmailFailed, err)
	}

	var plain strings.Builder
	var rich strings.Builder
	collectBodies(raw.Payload, &plain, &rich)

	text := plain.String()
	if strings.TrimSpace(text) == "" {
		text = stripHTML(rich.String())
	}

	message := model.EmailMessage{
		ID:          raw.ID,
		ThreadID:    raw.ThreadID,
		From:        headerValue(raw.Payload, "From"),
		To:          headerValue(raw.Payload, "To"),
		Subject:     headerValue(raw.Payload, "Subject"),
		Snippet:     raw.Snippet,
		Body:        truncate(text, c.maxBodyBytes),
		ReceivedAt:  parseInternalDate(raw.InternalDate),
		Attachments: collectAttachments(raw.Payload),
	}
	return message, nil
}

func (c *gmailClient) Attachment(ctx context.Context, accessToken string, messageID string, attachmentID string) ([]byte, error) {
	path := "/gmail/v1/users/me/messages/" + url.PathEscape(messageID) + "/attachments/" + url.PathEscape(attachmentID)
	body, status, err := c.requestWithLimit(ctx, http.MethodGet, path, accessToken, nil, gmailAttachmentLimit)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, gmailStatusError(status, body)
	}

	var raw struct {
		Data string `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("%w: giải mã attachment: %v", model.ErrGmailFailed, err)
	}
	decoded, ok := decodeBase64Bytes(raw.Data)
	if !ok {
		return nil, fmt.Errorf("%w: attachment không phải base64 hợp lệ", model.ErrGmailFailed)
	}
	return decoded, nil
}

func (c *gmailClient) ProfileHistoryID(ctx context.Context, accessToken string) (string, error) {
	body, status, err := c.request(ctx, http.MethodGet, "/gmail/v1/users/me/profile", accessToken, nil)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", gmailStatusError(status, body)
	}

	var raw struct {
		HistoryID string `json:"historyId"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("%w: giải mã profile: %v", model.ErrGmailFailed, err)
	}
	return raw.HistoryID, nil
}

func (c *gmailClient) request(ctx context.Context, method string, path string, accessToken string, payload any) ([]byte, int, error) {
	return c.requestWithLimit(ctx, method, path, accessToken, payload, gmailResponseLimit)
}

func (c *gmailClient) requestWithLimit(ctx context.Context, method string, path string, accessToken string, payload any, limit int64) ([]byte, int, error) {
	var reader io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, fmt.Errorf("%w: mã hoá request: %v", model.ErrGmailFailed, err)
		}
		reader = bytes.NewReader(encoded)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: tạo request: %v", model.ErrGmailFailed, err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Accept", "application/json")
	if payload != nil {
		request.Header.Set("Content-Type", "application/json")
	}

	response, err := c.http.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %v", model.ErrGmailFailed, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, limit))
	if err != nil {
		return nil, 0, fmt.Errorf("%w: đọc response: %v", model.ErrGmailFailed, err)
	}
	return body, response.StatusCode, nil
}

func gmailStatusError(status int, body []byte) error {
	switch status {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: gmail trả 401", model.ErrTokenRefresh)
	case http.StatusForbidden:
		return fmt.Errorf("%w: gmail trả 403 (thiếu scope hoặc hộp thư bị khoá)", model.ErrGmailFailed)
	default:
		return fmt.Errorf("%w: gmail trả %d: %s", model.ErrGmailFailed, status, truncate(string(body), 512))
	}
}

func collectBodies(payload *gmailPayload, plain *strings.Builder, rich *strings.Builder) {
	if payload == nil {
		return
	}
	switch payload.MimeType {
	case "text/plain":
		plain.WriteString(decodeBase64(payload.Body.Data))
	case "text/html":
		rich.WriteString(decodeBase64(payload.Body.Data))
	}
	for index := range payload.Parts {
		collectBodies(&payload.Parts[index], plain, rich)
	}
}

func collectAttachments(payload *gmailPayload) []model.EmailAttachment {
	out := make([]model.EmailAttachment, 0)
	var walk func(current *gmailPayload)
	walk = func(current *gmailPayload) {
		if current == nil {
			return
		}
		if current.Body.AttachmentID != "" {
			out = append(out, model.EmailAttachment{
				Filename:     strings.TrimSpace(current.Filename),
				MimeType:     current.MimeType,
				AttachmentID: current.Body.AttachmentID,
				Size:         current.Body.Size,
			})
		}
		for index := range current.Parts {
			walk(&current.Parts[index])
		}
	}
	walk(payload)
	return out
}

func headerValue(payload *gmailPayload, name string) string {
	if payload == nil {
		return ""
	}
	for _, header := range payload.Headers {
		if strings.EqualFold(header.Name, name) {
			return strings.TrimSpace(header.Value)
		}
	}
	return ""
}

func decodeBase64(value string) string {
	if value == "" {
		return ""
	}
	encodings := []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.StdEncoding}
	for _, encoding := range encodings {
		if decoded, err := encoding.DecodeString(value); err == nil {
			return string(decoded)
		}
	}
	return ""
}

func decodeBase64Bytes(value string) ([]byte, bool) {
	if value == "" {
		return nil, false
	}
	encodings := []*base64.Encoding{base64.RawURLEncoding, base64.URLEncoding, base64.StdEncoding}
	for _, encoding := range encodings {
		if decoded, err := encoding.DecodeString(value); err == nil {
			return decoded, true
		}
	}
	return nil, false
}

func stripHTML(value string) string {
	if value == "" {
		return ""
	}
	return strings.TrimSpace(whitespacePattern.ReplaceAllString(html.UnescapeString(htmlTagPattern.ReplaceAllString(value, " ")), " "))
}

func parseInternalDate(value string) time.Time {
	millis, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || millis <= 0 {
		return time.Time{}
	}
	return time.UnixMilli(millis).UTC()
}

func truncate(value string, limit int) string {
	if limit <= 0 || len(value) <= limit {
		return value
	}
	return value[:limit]
}
