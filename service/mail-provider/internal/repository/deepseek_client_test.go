package repository

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

func TestParseDraftReadsFencedJSON(t *testing.T) {
	content := "```json\n{\"is_actionable\": true, \"title\": \"Nộp báo cáo\", \"description\": \"Trước thứ sáu\", \"priority\": \"high\", \"due_at\": \"2026-01-02T09:00:00Z\"}\n```"

	draft, err := parseDraft(content)
	if err != nil {
		t.Fatalf("parse draft: %v", err)
	}
	if !draft.Actionable {
		t.Fatal("is_actionable=true phải được giữ")
	}
	if draft.Title != "Nộp báo cáo" || draft.Description != "Trước thứ sáu" {
		t.Fatalf("map title/description sai: %+v", draft)
	}
	if draft.Priority != model.PriorityHigh {
		t.Fatalf("priority phải là high, nhận %v", draft.Priority)
	}
	if draft.DueAt == nil || draft.DueAt.UTC().Format(time.RFC3339) != "2026-01-02T09:00:00Z" {
		t.Fatalf("due_at sai: %v", draft.DueAt)
	}
}

func TestParseDraftDefaultsActionableToTrue(t *testing.T) {
	draft, err := parseDraft(`{"title": "Việc"}`)
	if err != nil {
		t.Fatalf("parse draft: %v", err)
	}
	if !draft.Actionable {
		t.Fatal("thiếu is_actionable thì mặc định là true")
	}
	if draft.DueAt != nil {
		t.Fatalf("due_at rỗng phải là nil, nhận %v", draft.DueAt)
	}
}

func TestParseDraftMarksNonActionable(t *testing.T) {
	draft, err := parseDraft(`{"is_actionable": false, "title": "", "priority": ""}`)
	if err != nil {
		t.Fatalf("parse draft: %v", err)
	}
	if draft.Actionable {
		t.Fatal("is_actionable=false phải được giữ")
	}
	if draft.Priority != model.PriorityUnspecified {
		t.Fatalf("priority lạ phải là unspecified, nhận %v", draft.Priority)
	}
}

func TestParseDraftRejectsMissingJSON(t *testing.T) {
	if _, err := parseDraft("không có json ở đây"); !errors.Is(err, model.ErrAnalyzeFailed) {
		t.Fatalf("thiếu JSON phải là ErrAnalyzeFailed, nhận %v", err)
	}
	if _, err := parseDraft("{không phải json}"); !errors.Is(err, model.ErrAnalyzeFailed) {
		t.Fatalf("JSON hỏng phải là ErrAnalyzeFailed, nhận %v", err)
	}
}

func TestParsePriorityAliases(t *testing.T) {
	cases := map[string]model.Priority{
		"low":      model.PriorityLow,
		"normal":   model.PriorityMedium,
		"HIGH":     model.PriorityHigh,
		"critical": model.PriorityUrgent,
		"":         model.PriorityUnspecified,
	}
	for input, want := range cases {
		if got := parsePriority(input); got != want {
			t.Fatalf("parsePriority(%q) = %v, mong đợi %v", input, got, want)
		}
	}
}

func TestParseDueAtLayouts(t *testing.T) {
	cases := []string{"2026-01-02T09:00:00Z", "2026-01-02T09:00:00", "2026-01-02 09:00:00", "2026-01-02"}
	for _, input := range cases {
		if parsed := parseDueAt(input); parsed == nil {
			t.Fatalf("parseDueAt(%q) không được nil", input)
		}
	}
	if parsed := parseDueAt("không phải ngày"); parsed != nil {
		t.Fatalf("ngày không hợp lệ phải là nil, nhận %v", parsed)
	}
	if parsed := parseDueAt("   "); parsed != nil {
		t.Fatalf("chuỗi rỗng phải là nil, nhận %v", parsed)
	}
}

func TestDeepSeekAnalyzerRequiresAPIKey(t *testing.T) {
	analyzer := NewDeepSeekAnalyzer("", "http://127.0.0.1:1", "deepseek-chat", time.Second)
	if _, err := analyzer.Analyze(context.Background(), model.EmailMessage{}); !errors.Is(err, model.ErrNotConfigured) {
		t.Fatalf("thiếu API key phải là ErrNotConfigured, nhận %v", err)
	}
}

func TestDeepSeekAnalyzerSendsEmailAndParsesResponse(t *testing.T) {
	var captured map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Errorf("path sai: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer secret" {
			t.Errorf("authorization sai: %q", got)
		}
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Errorf("decode request: %v", err)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]string{
					"content": `{"is_actionable": true, "title": "Gọi khách hàng", "description": "Xác nhận hợp đồng", "priority": "urgent", "due_at": ""}`,
				},
			}},
		})
	}))
	defer server.Close()

	analyzer := NewDeepSeekAnalyzer("secret", server.URL, "deepseek-chat", 5*time.Second)
	draft, err := analyzer.Analyze(context.Background(), model.EmailMessage{
		From:    "boss@example.com",
		To:      "user@example.com",
		Subject: "Hợp đồng",
		Body:    "Gọi khách hàng xác nhận hợp đồng",
	})
	if err != nil {
		t.Fatalf("analyze: %v", err)
	}
	if draft.Title != "Gọi khách hàng" || draft.Priority != model.PriorityUrgent {
		t.Fatalf("draft sai: %+v", draft)
	}

	messages, ok := captured["messages"].([]any)
	if !ok || len(messages) != 2 {
		t.Fatalf("request phải có 2 message, nhận %#v", captured["messages"])
	}
	userMessage, _ := messages[1].(map[string]any)
	content, _ := userMessage["content"].(string)
	if !strings.Contains(content, "Hợp đồng") || !strings.Contains(content, "Gọi khách hàng") {
		t.Fatalf("prompt phải chứa nội dung email, nhận %q", content)
	}
}

func TestDeepSeekAnalyzerReportsHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	analyzer := NewDeepSeekAnalyzer("secret", server.URL, "deepseek-chat", 5*time.Second)
	if _, err := analyzer.Analyze(context.Background(), model.EmailMessage{}); !errors.Is(err, model.ErrAnalyzeFailed) {
		t.Fatalf("lỗi HTTP phải là ErrAnalyzeFailed, nhận %v", err)
	}
}
