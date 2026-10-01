package repository

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/oauth2"

	"taskmanager/service/mail-provider/internal/model"
)

func newTestPullClient(baseURL string, subscription string) *pubsubPullClient {
	client := NewPubSubPullClient(baseURL, subscription, 5, 5*time.Second)
	client.tokens = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "test-token"})
	return client
}

func TestPubSubPullClientReceiveDecodesNotices(t *testing.T) {
	valid := base64.StdEncoding.EncodeToString([]byte(`{"emailAddress":"User@Example.com","historyId":"42"}`))
	missingEmail := base64.StdEncoding.EncodeToString([]byte(`{"historyId":"1"}`))
	numericHistoryID := base64.StdEncoding.EncodeToString([]byte(`{"emailAddress":"num@example.com","historyId":998877}`))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/demo/subscriptions/gmail-pull:pull" {
			t.Errorf("path sai: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Errorf("authorization sai: %q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		if body["returnImmediately"] != false {
			t.Errorf("returnImmediately phải là false, nhận %v", body["returnImmediately"])
		}
		if body["maxMessages"] != float64(5) {
			t.Errorf("maxMessages sai: %v", body["maxMessages"])
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"receivedMessages": []map[string]any{
				{"ackId": "ack-1", "message": map[string]string{"data": valid, "messageId": "m1"}},
				{"ackId": "ack-2", "message": map[string]string{"data": missingEmail}},
				{"ackId": "ack-3", "message": map[string]string{"data": "!!!không-phải-base64!!!"}},
				{"ackId": "ack-4", "message": map[string]string{"data": numericHistoryID, "messageId": "m4"}},
			},
		})
	}))
	defer server.Close()

	client := newTestPullClient(server.URL, "projects/demo/subscriptions/gmail-pull")
	notices, err := client.Receive(context.Background())
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if len(notices) != 4 {
		t.Fatalf("phải nhận 4 message, nhận %d", len(notices))
	}

	first := notices[0]
	if first.AckID != "ack-1" || first.Notice.EmailAddress != "user@example.com" || first.Notice.HistoryID != "42" || first.Notice.MessageID != "m1" {
		t.Fatalf("notice đầu sai: %+v", first)
	}
	if notices[1].AckID != "ack-2" || notices[1].Notice.EmailAddress != "" || notices[1].Raw == "" {
		t.Fatalf("notice thiếu email phải giữ ackId, notice rỗng và Raw để log: %+v", notices[1])
	}
	if notices[2].AckID != "ack-3" || notices[2].Notice.EmailAddress != "" || notices[2].Raw == "" {
		t.Fatalf("notice base64 hỏng phải giữ ackId, notice rỗng và Raw để log: %+v", notices[2])
	}
	if notices[3].Notice.EmailAddress != "num@example.com" || notices[3].Notice.HistoryID != "998877" {
		t.Fatalf("historyId dạng số phải parse được qua pull: %+v", notices[3])
	}
}

func TestPubSubPullClientReceiveEmptyResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := newTestPullClient(server.URL, "projects/demo/subscriptions/gmail-pull")
	notices, err := client.Receive(context.Background())
	if err != nil {
		t.Fatalf("receive: %v", err)
	}
	if len(notices) != 0 {
		t.Fatalf("response rỗng phải trả 0 notice, nhận %d", len(notices))
	}
}

func TestPubSubPullClientAcknowledge(t *testing.T) {
	var got []string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/projects/demo/subscriptions/gmail-pull:acknowledge" {
			t.Errorf("path sai: %s", r.URL.Path)
		}
		var body struct {
			AckIDs []string `json:"ackIds"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode body: %v", err)
		}
		got = body.AckIDs
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client := newTestPullClient(server.URL, "projects/demo/subscriptions/gmail-pull")
	if err := client.Acknowledge(context.Background(), []string{"a", "b"}); err != nil {
		t.Fatalf("acknowledge: %v", err)
	}
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("ackIds sai: %v", got)
	}
}

func TestPubSubPullClientAcknowledgeNoopWhenEmpty(t *testing.T) {
	client := NewPubSubPullClient("http://127.0.0.1:1", "projects/demo/subscriptions/gmail-pull", 5, time.Second)
	if err := client.Acknowledge(context.Background(), nil); err != nil {
		t.Fatalf("ack rỗng phải là no-op, nhận %v", err)
	}
}

func TestPubSubPullClientRequiresSubscription(t *testing.T) {
	client := NewPubSubPullClient("http://127.0.0.1:1", "   ", 5, time.Second)
	if _, err := client.Receive(context.Background()); !errors.Is(err, model.ErrNotConfigured) {
		t.Fatalf("thiếu subscription phải là ErrNotConfigured, nhận %v", err)
	}
}

func TestPubSubPullClientReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":{"status":"PERMISSION_DENIED"}}`))
	}))
	defer server.Close()

	client := newTestPullClient(server.URL, "projects/demo/subscriptions/gmail-pull")
	if _, err := client.Receive(context.Background()); err == nil {
		t.Fatal("403 phải trả lỗi")
	}
}
