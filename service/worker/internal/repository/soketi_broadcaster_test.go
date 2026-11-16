package repository

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskmanager/service/worker/internal/model"
)

func TestBroadcastUpdatedSendsCardToBothChannels(t *testing.T) {
	var body []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ = io.ReadAll(r.Body)
		_, _ = w.Write([]byte("{}"))
	}))
	defer server.Close()

	message := model.Message{ID: "m1", SenderID: "alice", RecipientID: "bob", CreatedAt: time.Now(),
		Task: &model.TaskRef{TaskID: "t1", StatusName: "Done", StatusCategory: "done", StatusChangedAt: time.Now()}}
	if err := NewSoketiBroadcaster(server.URL, "id", "key", "secret", "chat-", time.Second).BroadcastUpdated(context.Background(), message); err != nil {
		t.Fatalf("broadcast: %v", err)
	}

	var event soketiEvent
	if err := json.Unmarshal(body, &event); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if event.Name != "message.updated" || strings.Join(event.Channels, ",") != "private-chat-bob,private-chat-alice" {
		t.Fatalf("event sai: %+v", event)
	}
	var payload struct {
		ID   string `json:"id"`
		Task struct {
			StatusCategory string `json:"statusCategory"`
		} `json:"task"`
	}
	if err := json.Unmarshal([]byte(event.Data), &payload); err != nil || payload.ID != "m1" || payload.Task.StatusCategory != "done" {
		t.Fatalf("payload sai: %s", event.Data)
	}
}
