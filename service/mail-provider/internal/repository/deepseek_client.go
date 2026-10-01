package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

const analyzerSystemPrompt = `Bạn là trợ lý chuyển email thành công việc cá nhân.
Đọc email người dùng nhận được và quyết định xem nó có cần tạo công việc hay không.
Chỉ trả về DUY NHẤT một JSON object, không kèm giải thích, không bọc code fence, đúng schema:
{"is_actionable": true, "title": "tiêu đề ngắn gọn", "description": "mô tả chi tiết", "priority": "low|medium|high|urgent", "due_at": "RFC3339 hoặc chuỗi rỗng"}
Quy tắc:
- is_actionable = false nếu email chỉ là thông báo, quảng cáo, newsletter hoặc không cần hành động.
- title tối đa 120 ký tự, bắt đầu bằng động từ hành động.
- description tóm tắt việc cần làm, kèm thông tin hữu ích (người gửi, deadline, link nếu có).
- due_at chỉ điền khi email nêu rõ hạn; nếu không rõ thì để chuỗi rỗng.`

type deepSeekAnalyzer struct {
	apiKey  string
	baseURL string
	model   string
	http    *http.Client
}

func NewDeepSeekAnalyzer(apiKey string, baseURL string, model string, timeout time.Duration) *deepSeekAnalyzer {
	return &deepSeekAnalyzer{
		apiKey:  apiKey,
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   model,
		http:    &http.Client{Timeout: timeout},
	}
}

func (a *deepSeekAnalyzer) Analyze(ctx context.Context, message model.EmailMessage) (model.TaskDraft, error) {
	if strings.TrimSpace(a.apiKey) == "" {
		return model.TaskDraft{}, model.ErrNotConfigured
	}

	payload := map[string]any{
		"model": a.model,
		"messages": []map[string]string{
			{"role": "system", "content": analyzerSystemPrompt},
			{"role": "user", "content": emailPrompt(message)},
		},
		"response_format": map[string]string{"type": "json_object"},
		"temperature":     0,
		"stream":          false,
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return model.TaskDraft{}, fmt.Errorf("%w: mã hoá request: %v", model.ErrAnalyzeFailed, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return model.TaskDraft{}, fmt.Errorf("%w: tạo request: %v", model.ErrAnalyzeFailed, err)
	}
	request.Header.Set("Authorization", "Bearer "+a.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := a.http.Do(request)
	if err != nil {
		return model.TaskDraft{}, fmt.Errorf("%w: %v", model.ErrAnalyzeFailed, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return model.TaskDraft{}, fmt.Errorf("%w: đọc response: %v", model.ErrAnalyzeFailed, err)
	}
	if response.StatusCode != http.StatusOK {
		return model.TaskDraft{}, fmt.Errorf("%w: deepseek trả %d: %s", model.ErrAnalyzeFailed, response.StatusCode, truncate(string(body), 512))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &completion); err != nil {
		return model.TaskDraft{}, fmt.Errorf("%w: giải mã response: %v", model.ErrAnalyzeFailed, err)
	}
	if len(completion.Choices) == 0 {
		return model.TaskDraft{}, fmt.Errorf("%w: deepseek không trả lựa chọn nào", model.ErrAnalyzeFailed)
	}

	return parseDraft(completion.Choices[0].Message.Content)
}

func emailPrompt(message model.EmailMessage) string {
	var builder strings.Builder
	builder.WriteString("From: ")
	builder.WriteString(message.From)
	builder.WriteString("\nTo: ")
	builder.WriteString(message.To)
	builder.WriteString("\nSubject: ")
	builder.WriteString(message.Subject)
	if !message.ReceivedAt.IsZero() {
		builder.WriteString("\nReceived: ")
		builder.WriteString(message.ReceivedAt.Format(time.RFC3339))
	}
	builder.WriteString("\n\n")
	builder.WriteString(message.Body)
	return builder.String()
}

func parseDraft(content string) (model.TaskDraft, error) {
	jsonPayload := extractJSONObject(content)
	if jsonPayload == "" {
		return model.TaskDraft{}, fmt.Errorf("%w: không tìm thấy JSON trong phản hồi", model.ErrAnalyzeFailed)
	}

	var raw struct {
		Actionable  *bool  `json:"is_actionable"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		DueAt       string `json:"due_at"`
	}
	if err := json.Unmarshal([]byte(jsonPayload), &raw); err != nil {
		return model.TaskDraft{}, fmt.Errorf("%w: JSON không hợp lệ: %v", model.ErrAnalyzeFailed, err)
	}

	actionable := true
	if raw.Actionable != nil {
		actionable = *raw.Actionable
	}

	return model.TaskDraft{
		Actionable:  actionable,
		Title:       strings.TrimSpace(raw.Title),
		Description: strings.TrimSpace(raw.Description),
		Priority:    parsePriority(raw.Priority),
		DueAt:       parseDueAt(raw.DueAt),
	}, nil
}

func extractJSONObject(content string) string {
	start := strings.Index(content, "{")
	end := strings.LastIndex(content, "}")
	if start < 0 || end <= start {
		return ""
	}
	return content[start : end+1]
}

func parsePriority(value string) model.Priority {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "low":
		return model.PriorityLow
	case "medium", "normal":
		return model.PriorityMedium
	case "high":
		return model.PriorityHigh
	case "urgent", "critical":
		return model.PriorityUrgent
	default:
		return model.PriorityUnspecified
	}
}

func parseDueAt(value string) *time.Time {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	layouts := []string{time.RFC3339, "2006-01-02T15:04:05", "2006-01-02 15:04:05", "2006-01-02"}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			utc := parsed.UTC()
			return &utc
		}
	}
	return nil
}
