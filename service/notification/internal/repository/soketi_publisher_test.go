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

	"taskmanager/service/notification/internal/model"
)

func TestSoketiPublisherPublishSignsRequest(t *testing.T) {
	var gotPath string
	var gotQuery string
	var gotBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	publisher := NewSoketiPublisher(server.URL, "app-id", "app-key", "app-secret", "noti-internal-", 5*time.Second)
	notice := model.Notice{
		ID:         "notice-1",
		ProfileID:  "profile-1",
		Type:       model.TypeTaskCreated,
		Title:      "Nộp báo cáo",
		TargetType: model.TargetTypeTask,
		TargetID:   "task-1",
		CreatedAt:  time.Now().UTC(),
	}

	if err := publisher.Publish(context.Background(), notice); err != nil {
		t.Fatalf("Publish lỗi: %v", err)
	}

	if gotPath != "/apps/app-id/events" {
		t.Fatalf("path = %q, muốn /apps/app-id/events", gotPath)
	}

	var event soketiEvent
	if err := json.Unmarshal(gotBody, &event); err != nil {
		t.Fatalf("decode event: %v", err)
	}
	if event.Name != soketiEventName {
		t.Fatalf("event name = %q, muốn %q", event.Name, soketiEventName)
	}
	if len(event.Channels) != 1 || event.Channels[0] != "private-noti-internal-profile-1" {
		t.Fatalf("channels = %+v", event.Channels)
	}

	var payload soketiNotice
	if err := json.Unmarshal([]byte(event.Data), &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.ID != "notice-1" || payload.TargetType != model.TargetTypeTask {
		t.Fatalf("payload sai: %+v", payload)
	}

	values, err := url.ParseQuery(gotQuery)
	if err != nil {
		t.Fatalf("parse query: %v", err)
	}
	if values.Get("auth_key") != "app-key" || values.Get("auth_version") != "1.0" {
		t.Fatalf("thiếu tham số auth: %s", gotQuery)
	}

	sum := md5.Sum(gotBody)
	if values.Get("body_md5") != hex.EncodeToString(sum[:]) {
		t.Fatalf("body_md5 sai: %q", values.Get("body_md5"))
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
		t.Fatalf("auth_signature sai: %q", signature)
	}
}

func TestSoketiPublisherPublishReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("bad auth"))
	}))
	defer server.Close()

	publisher := NewSoketiPublisher(server.URL, "app-id", "app-key", "app-secret", "noti-internal-", 5*time.Second)
	err := publisher.Publish(context.Background(), model.Notice{ProfileID: "profile-1"})
	if err == nil {
		t.Fatal("Publish phải trả lỗi khi soketi trả 401")
	}
}

func TestSoketiPublisherRequiresConfiguration(t *testing.T) {
	publisher := NewSoketiPublisher("", "", "", "", "noti-internal-", 0)
	if publisher.Configured() {
		t.Fatal("publisher rỗng không được coi là configured")
	}
	if err := publisher.Publish(context.Background(), model.Notice{ProfileID: "p"}); err == nil {
		t.Fatal("Publish phải trả lỗi khi chưa cấu hình")
	}
}
