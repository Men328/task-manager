package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

func TestGmailWatchParsesExpiration(t *testing.T) {
	expiration := time.Now().Add(6 * 24 * time.Hour).UnixMilli()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/watch" {
			t.Errorf("path sai: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer token" {
			t.Errorf("authorization sai: %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["topicName"] != "projects/demo/topics/gmail" {
			t.Errorf("topicName sai: %v", body["topicName"])
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"historyId":  "77",
			"expiration": strconv.FormatInt(expiration, 10),
		})
	}))
	defer server.Close()

	client := NewGmailClient(server.URL, 5*time.Second, 1024)
	result, err := client.Watch(context.Background(), "token", "projects/demo/topics/gmail", []string{"INBOX"})
	if err != nil {
		t.Fatalf("watch: %v", err)
	}
	if result.HistoryID != "77" {
		t.Fatalf("historyId sai: %q", result.HistoryID)
	}
	if result.ExpiresAt.UnixMilli() != expiration {
		t.Fatalf("expiration sai: %s", result.ExpiresAt)
	}
}

func TestGmailHistoryMapsMessageIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/history" {
			t.Errorf("path sai: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("startHistoryId"); got != "100" {
			t.Errorf("startHistoryId sai: %q", got)
		}
		if got := r.URL.Query().Get("historyTypes"); got != "messageAdded" {
			t.Errorf("historyTypes sai: %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"history": []map[string]any{
				{"messages": []map[string]string{{"id": "a"}, {"id": "b"}}},
				{"messages": []map[string]string{{"id": "a"}}},
			},
			"historyId": "200",
		})
	}))
	defer server.Close()

	client := NewGmailClient(server.URL, 5*time.Second, 1024)
	ids, latest, err := client.NewMessageIDs(context.Background(), "token", "100", []string{"INBOX"})
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if latest != "200" {
		t.Fatalf("historyId mới sai: %q", latest)
	}
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Fatalf("phải khử trùng lặp và giữ thứ tự, nhận %v", ids)
	}
}

func TestGmailHistoryDetectsExpiredCheckpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":404}}`))
	}))
	defer server.Close()

	client := NewGmailClient(server.URL, 5*time.Second, 1024)
	if _, _, err := client.NewMessageIDs(context.Background(), "token", "1", nil); !errors.Is(err, model.ErrHistoryGone) {
		t.Fatalf("404 phải là ErrHistoryGone, nhận %v", err)
	}
}

func TestGmailHistoryDetectsExpiredToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewGmailClient(server.URL, 5*time.Second, 1024)
	if _, _, err := client.NewMessageIDs(context.Background(), "token", "1", nil); !errors.Is(err, model.ErrTokenRefresh) {
		t.Fatalf("401 phải là ErrTokenRefresh, nhận %v", err)
	}
}

func TestGmailNewMessageIDsWithoutCheckpointUsesProfile(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/gmail/v1/users/me/profile" {
			t.Errorf("path sai: %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"historyId": "321"})
	}))
	defer server.Close()

	client := NewGmailClient(server.URL, 5*time.Second, 1024)
	ids, latest, err := client.NewMessageIDs(context.Background(), "token", "", nil)
	if err != nil {
		t.Fatalf("history không checkpoint: %v", err)
	}
	if len(ids) != 0 || latest != "321" {
		t.Fatalf("phải lấy historyId hiện tại, nhận %v / %q", ids, latest)
	}
}

func TestGmailMessageExtractsPlainTextBody(t *testing.T) {
	encoded := base64.RawURLEncoding.EncodeToString([]byte("Nội dung thư"))
	internalDate := time.Now().UTC().Truncate(time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/gmail/v1/users/me/messages/") {
			t.Errorf("path sai: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("format"); got != "full" {
			t.Errorf("format sai: %q", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":           "m1",
			"threadId":     "t1",
			"snippet":      "snip",
			"internalDate": strconv.FormatInt(internalDate.UnixMilli(), 10),
			"payload": map[string]any{
				"mimeType": "multipart/alternative",
				"headers": []map[string]string{
					{"name": "From", "value": "boss@example.com"},
					{"name": "To", "value": "user@example.com"},
					{"name": "Subject", "value": "Họp nhóm"},
				},
				"parts": []map[string]any{
					{"mimeType": "text/html", "body": map[string]string{"data": base64.RawURLEncoding.EncodeToString([]byte("<p>html</p>"))}},
					{"mimeType": "text/plain", "body": map[string]string{"data": encoded}},
				},
			},
		})
	}))
	defer server.Close()

	client := NewGmailClient(server.URL, 5*time.Second, 4096)
	message, err := client.Message(context.Background(), "token", "m1")
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	if message.From != "boss@example.com" || message.Subject != "Họp nhóm" {
		t.Fatalf("header sai: %+v", message)
	}
	if message.Body != "Nội dung thư" {
		t.Fatalf("phải ưu tiên text/plain, nhận %q", message.Body)
	}
	if !message.ReceivedAt.Equal(internalDate) {
		t.Fatalf("internalDate sai: %s != %s", message.ReceivedAt, internalDate)
	}
}

func TestGmailMessageFallsBackToStrippedHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "m1",
			"payload": map[string]any{
				"mimeType": "text/html",
				"body":     map[string]string{"data": base64.RawURLEncoding.EncodeToString([]byte("<p>Xin   chào</p><br/>bạn"))},
			},
		})
	}))
	defer server.Close()

	client := NewGmailClient(server.URL, 5*time.Second, 4096)
	message, err := client.Message(context.Background(), "token", "m1")
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	if message.Body != "Xin chào bạn" {
		t.Fatalf("phải strip HTML, nhận %q", message.Body)
	}
}

func TestGmailMessageTruncatesBody(t *testing.T) {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(strings.Repeat("x", 100)))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "m1",
			"payload": map[string]any{
				"mimeType": "text/plain",
				"body":     map[string]string{"data": encoded},
			},
		})
	}))
	defer server.Close()

	client := NewGmailClient(server.URL, 5*time.Second, 10)
	message, err := client.Message(context.Background(), "token", "m1")
	if err != nil {
		t.Fatalf("message: %v", err)
	}
	if len(message.Body) != 10 {
		t.Fatalf("body phải bị cắt còn 10 byte, nhận %d", len(message.Body))
	}
}

func TestGmailStopPropagatesFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewGmailClient(server.URL, 5*time.Second, 1024)
	if err := client.Stop(context.Background(), "token"); !errors.Is(err, model.ErrGmailFailed) {
		t.Fatalf("403 phải là ErrGmailFailed, nhận %v", err)
	}
}

func TestDecodeBase64AcceptsURLAndPaddedVariants(t *testing.T) {
	value := "nội dung"
	raw := []byte(value)
	for _, encoded := range []string{
		base64.RawURLEncoding.EncodeToString(raw),
		base64.URLEncoding.EncodeToString(raw),
		base64.StdEncoding.EncodeToString(raw),
		base64.RawStdEncoding.EncodeToString(raw),
	} {
		if got := decodeBase64(encoded); got != value {
			t.Fatalf("decodeBase64(%q) = %q, mong đợi %q", encoded, got, value)
		}
	}
	if got := decodeBase64(""); got != "" {
		t.Fatalf("chuỗi rỗng phải trả rỗng, nhận %q", got)
	}
	if got := decodeBase64("!!!not-base64!!!"); got != "" {
		t.Fatalf("base64 hỏng phải trả rỗng, nhận %q", got)
	}
}

func TestStripHTMLRemovesTagsAndCollapsesWhitespace(t *testing.T) {
	if got := stripHTML("<div>Xin   <b>chào</b></div>\n\n  bạn"); got != "Xin chào bạn" {
		t.Fatalf("stripHTML sai: %q", got)
	}
	if got := stripHTML(""); got != "" {
		t.Fatalf("chuỗi rỗng phải trả rỗng, nhận %q", got)
	}
}

func TestParseInternalDateRejectsInvalid(t *testing.T) {
	if got := parseInternalDate("không phải số"); !got.IsZero() {
		t.Fatalf("internalDate hỏng phải là zero time, nhận %s", got)
	}
	if got := parseInternalDate(""); !got.IsZero() {
		t.Fatalf("internalDate rỗng phải là zero time, nhận %s", got)
	}
}
