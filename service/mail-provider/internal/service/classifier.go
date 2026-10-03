package service

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

var eventKeywords = []string{
	"sự kiện", "hội thảo", "hội nghị", "conference", "webinar", "workshop",
	"seminar", "sinh nhật", "birthday", "đám cưới", "wedding", "tiệc", "party",
	"khai trương", "ra mắt", "lễ", "ceremony", "thiệp mời", "invitation",
	"mời bạn", "mời tham dự", "mời tham gia", "event",
}

var scheduleKeywords = []string{
	"lịch", "lịch hẹn", "appointment", "schedule", "calendar", "cuộc họp",
	"meeting", "họp", "đặt phòng", "booking", "reminder", "nhắc lịch",
	"ca làm", "shift", "call", "zoom", "google meet",
}

var taskKeywords = []string{
	"cần", "hoàn thành", "nộp", "thanh toán", "hóa đơn", "hoá đơn", "invoice",
	"deadline", "hạn chót", "yêu cầu", "action required", "todo", "to-do",
	"task", "công việc", "xử lý", "kiểm tra", "duyệt", "review", "báo cáo",
	"xác nhận",
}

var isoTimePattern = regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})(?:[ t](\d{1,2}):(\d{2}))?`)

var dayFirstTimePattern = regexp.MustCompile(`\b(\d{1,2})[/](\d{1,2})(?:[/](\d{2,4}))?(?:\s+(\d{1,2}):(\d{2}))?`)

func classifyByRules(message model.EmailMessage) model.MailDraft {
	subject := strings.ToLower(strings.TrimSpace(message.Subject))
	text := strings.ToLower(strings.Join([]string{message.Subject, message.Snippet, message.Body}, "\n"))

	eventScore := keywordScore(text, subject, eventKeywords)
	scheduleScore := keywordScore(text, subject, scheduleKeywords)
	taskScore := keywordScore(text, subject, taskKeywords)

	category := model.CategoryOther
	best := 0
	if eventScore > best {
		category, best = model.CategoryEvent, eventScore
	}
	if scheduleScore > best {
		category, best = model.CategorySchedule, scheduleScore
	}
	if taskScore > best {
		category, best = model.CategoryTask, taskScore
	}

	draft := model.MailDraft{
		Category:    category,
		Actionable:  category != model.CategoryOther,
		Title:       draftTitle(message),
		Description: draftDescription(message),
	}
	if category == model.CategoryOther {
		draft.Reason = "no_rule_matched"
		return draft
	}
	if category == model.CategorySchedule || category == model.CategoryEvent {
		draft.StartAt = parseFirstTime(text)
	}
	return draft
}

func keywordScore(text string, subject string, keywords []string) int {
	score := 0
	for _, keyword := range keywords {
		if !strings.Contains(text, keyword) {
			continue
		}
		score++
		if strings.Contains(subject, keyword) {
			score++
		}
	}
	return score
}

func draftTitle(message model.EmailMessage) string {
	if trimmed := strings.TrimSpace(message.Subject); trimmed != "" {
		return truncateRunes(trimmed, 120)
	}
	for _, line := range strings.Split(message.Body, "\n") {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			return truncateRunes(trimmed, 120)
		}
	}
	return "Email không có tiêu đề"
}

func draftDescription(message model.EmailMessage) string {
	body := strings.TrimSpace(message.Body)
	if body == "" {
		body = strings.TrimSpace(message.Snippet)
	}
	return truncateRunes(body, 500)
}

func parseFirstTime(text string) *time.Time {
	if match := isoTimePattern.FindStringSubmatch(text); match != nil {
		year := atoi(match[1])
		month := atoi(match[2])
		day := atoi(match[3])
		hour := atoi(match[4])
		minute := atoi(match[5])
		if validClock(hour, minute) && validDate(year, month, day) {
			parsed := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
			return &parsed
		}
	}

	if match := dayFirstTimePattern.FindStringSubmatch(text); match != nil {
		day := atoi(match[1])
		month := atoi(match[2])
		year := time.Now().UTC().Year()
		if match[3] != "" {
			year = atoi(match[3])
			if year < 100 {
				year += 2000
			}
		}
		hour := atoi(match[4])
		minute := atoi(match[5])
		if validClock(hour, minute) && validDate(year, month, day) {
			parsed := time.Date(year, time.Month(month), day, hour, minute, 0, 0, time.UTC)
			return &parsed
		}
	}
	return nil
}

func validClock(hour int, minute int) bool {
	return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
}

func validDate(year int, month int, day int) bool {
	if year < 1970 || year > 2200 || month < 1 || month > 12 || day < 1 || day > 31 {
		return false
	}
	return true
}

func atoi(value string) int {
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
