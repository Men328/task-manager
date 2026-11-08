package repository

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strings"
	"testing"
	"time"

	"taskmanager/service/message/internal/model"
)

func TestSoketiBroadcasterSendsToBothPrivateChannelsAndSigns(t *testing.T) {
	var gotPath, gotQuery string
	var gotBody []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		gotBody, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	broadcaster := NewSoketiBroadcaster(server.URL, "app-id", "app-key", "app-secret", "chat-", time.Second)
	createdAt := time.Date(2026, 11, 8, 9, 30, 0, 123456000, time.UTC)
	message := model.Message{ID: "m1", ConversationID: "c1", SenderID: "alice", RecipientID: "bob", Body: "chào", ClientID: "tmp-1", CreatedAt: createdAt}

	if err := broadcaster.Broadcast(context.Background(), message); err != nil {
		t.Fatalf("broadcast: %v", err)
	}
	if gotPath != "/apps/app-id/events" {
		t.Fatalf("path sai: %s", gotPath)
	}

	var event soketiEvent
	if err := json.Unmarshal(gotBody, &event); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if event.Name != MessageEventName || strings.Join(event.Channels, ",") != "private-chat-bob,private-chat-alice" {
		t.Fatalf("event sai: %+v", event)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload["conversationId"] != "c1" || payload["clientId"] != "tmp-1" || payload["createdAt"] != "2026-11-08T09:30:00.123456Z" {
		t.Fatalf("payload phải theo camelCase như API, nhận %+v", payload)
	}

	values, _ := url.ParseQuery(gotQuery)
	sum := md5.Sum(gotBody)
	if values.Get("body_md5") != hex.EncodeToString(sum[:]) {
		t.Fatal("body_md5 sai")
	}
	signature := values.Get("auth_signature")
	values.Del("auth_signature")
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+values.Get(key))
	}
	mac := hmac.New(sha256.New, []byte("app-secret"))
	mac.Write([]byte("POST\n/apps/app-id/events\n" + strings.Join(parts, "&")))
	if signature != hex.EncodeToString(mac.Sum(nil)) {
		t.Fatal("chữ ký pusher sai")
	}
}

func TestSoketiBroadcasterReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	err := NewSoketiBroadcaster(server.URL, "id", "key", "secret", "chat-", time.Second).
		Broadcast(context.Background(), model.Message{SenderID: "a", RecipientID: "b"})
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("phải báo lỗi HTTP 401, nhận %v", err)
	}
}
