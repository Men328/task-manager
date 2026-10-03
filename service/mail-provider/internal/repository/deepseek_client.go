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

const analyzerSystemPrompt = `Bạn là trợ lý phân loại email cho một ứng dụng quản lý công việc cá nhân.
Đọc email người dùng nhận được và chọn ĐÚNG MỘT nhóm:
- "task": việc cần làm (yêu cầu hành động, deadline, thanh toán, báo cáo...).
- "schedule": lịch hẹn / lịch trình có thời gian cụ thể (cuộc họp, appointment, ca làm...).
- "event": sự kiện / thiệp mời / hội thảo / tiệc / lễ (có thể có thời gian, thường cần xác nhận).
- "other": thông báo, quảng cáo, newsletter, spam hoặc không đủ dữ kiện để xếp 3 nhóm trên.
Chỉ trả về DUY NHẤT một JSON object, không kèm giải thích, không bọc code fence, đúng schema:
{"category":"task|schedule|event|other","is_actionable":true,"title":"tiêu đề ngắn gọn","description":"mô tả chi tiết","priority":"low|medium|high|urgent","due_at":"RFC3339 hoặc chuỗi rỗng","start_at":"RFC3339 hoặc chuỗi rỗng","end_at":"RFC3339 hoặc chuỗi rỗng","all_day":false,"location":"địa điểm hoặc chuỗi rỗng","reason":"lý do ngắn nếu là other"}
Quy tắc:
- category = "other" khi email không thuộc task/schedule/event; khi đó is_actionable = false và điền reason.
- is_actionable = false với thông báo, quảng cáo, newsletter.
- title tối đa 120 ký tự. Với task bắt đầu bằng động từ hành động.
- description tóm tắt nội dung hữu ích (người gửi, deadline, link nếu có).
- start_at/end_at chỉ điền khi email nêu rõ thời gian; không rõ thì để chuỗi rỗng.
- due_at chỉ điền khi email nêu rõ hạn; không rõ thì để chuỗi rỗng.`

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

func (a *deepSeekAnalyzer) Analyze(ctx context.Context, message model.EmailMessage) (model.MailDraft, error) {
	if strings.TrimSpace(a.apiKey) == "" {
		return model.MailDraft{}, model.ErrNotConfigured
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
		return model.MailDraft{}, fmt.Errorf("%w: mã hoá request: %v", model.ErrAnalyzeFailed, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, a.baseURL+"/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return model.MailDraft{}, fmt.Errorf("%w: tạo request: %v", model.ErrAnalyzeFailed, err)
	}
	request.Header.Set("Authorization", "Bearer "+a.apiKey)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json")

	response, err := a.http.Do(request)
	if err != nil {
		return model.MailDraft{}, fmt.Errorf("%w: %v", model.ErrAnalyzeFailed, err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return model.MailDraft{}, fmt.Errorf("%w: đọc response: %v", model.ErrAnalyzeFailed, err)
	}
	if response.StatusCode != http.StatusOK {
		return model.MailDraft{}, fmt.Errorf("%w: deepseek trả %d: %s", model.ErrAnalyzeFailed, response.StatusCode, truncate(string(body), 512))
	}

	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(body, &completion); err != nil {
		return model.MailDraft{}, fmt.Errorf("%w: giải mã response: %v", model.ErrAnalyzeFailed, err)
	}
	if len(completion.Choices) == 0 {
		return model.MailDraft{}, fmt.Errorf("%w: deepseek không trả lựa chọn nào", model.ErrAnalyzeFailed)
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

func parseDraft(content string) (model.MailDraft, error) {
	jsonPayload := extractJSONObject(content)
	if jsonPayload == "" {
		return model.MailDraft{}, fmt.Errorf("%w: không tìm thấy JSON trong phản hồi", model.ErrAnalyzeFailed)
	}

	var raw struct {
		Category    string `json:"category"`
		Actionable  *bool  `json:"is_actionable"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Priority    string `json:"priority"`
		DueAt       string `json:"due_at"`
		StartAt     string `json:"start_at"`
		EndAt       string `json:"end_at"`
		AllDay      bool   `json:"all_day"`
		Location    string `json:"location"`
		Reason      string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(jsonPayload), &raw); err != nil {
		return model.MailDraft{}, fmt.Errorf("%w: JSON không hợp lệ: %v", model.ErrAnalyzeFailed, err)
	}

	actionable := true
	if raw.Actionable != nil {
		actionable = *raw.Actionable
	}

	return model.MailDraft{
		Category:    parseCategory(raw.Category),
		Actionable:  actionable,
		Title:       strings.TrimSpace(raw.Title),
		Description: strings.TrimSpace(raw.Description),
		Priority:    parsePriority(raw.Priority),
		DueAt:       parseDueAt(raw.DueAt),
		StartAt:     parseDueAt(raw.StartAt),
		EndAt:       parseDueAt(raw.EndAt),
		AllDay:      raw.AllDay,
		Location:    strings.TrimSpace(raw.Location),
		Reason:      strings.TrimSpace(raw.Reason),
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

func parseCategory(value string) model.Category {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(model.CategoryTask):
		return model.CategoryTask
	case string(model.CategorySchedule):
		return model.CategorySchedule
	case string(model.CategoryEvent):
		return model.CategoryEvent
	case string(model.CategoryOther):
		return model.CategoryOther
	default:
		return ""
	}
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
