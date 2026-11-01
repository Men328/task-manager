package service

import (
	"strings"
	"time"

	"github.com/teambition/rrule-go"

	"taskmanager/service/task/internal/model"
)

const (
	defaultRecurrenceTZ = "Asia/Ho_Chi_Minh"
	maxRecurrenceGap    = 365
)

func invalidRecurrence(format string, args ...any) error {
	return model.NewError(model.ErrorKindRecurrenceInvalid, format, args...)
}

func normalizeRecurrence(rule, tz string, due *time.Time) (string, string, error) {
	rule = strings.ToUpper(strings.TrimSpace(rule))
	if rule == "" {
		return "", "", nil
	}
	if due == nil {
		return "", "", invalidRecurrence("task lặp cần có hạn")
	}
	opt, err := parseRecurrence(rule)
	if err != nil {
		return "", "", err
	}
	tz = strings.TrimSpace(tz)
	if tz == "" {
		tz = defaultRecurrenceTZ
	}
	if _, err := time.LoadLocation(tz); err != nil {
		return "", "", invalidRecurrence("múi giờ không hợp lệ: %s", tz)
	}
	return opt.RRuleString(), tz, nil
}

func parseRecurrence(rule string) (*rrule.ROption, error) {
	opt, err := rrule.StrToROption(strings.ToUpper(rule))
	if err != nil {
		return nil, invalidRecurrence("quy tắc lặp không đọc được: %v", err)
	}
	switch opt.Freq {
	case rrule.DAILY, rrule.WEEKLY, rrule.MONTHLY, rrule.YEARLY:
	default:
		return nil, invalidRecurrence("chỉ hỗ trợ lặp theo ngày, tuần, tháng, năm")
	}
	if opt.Interval < 0 || opt.Interval > maxRecurrenceGap {
		return nil, invalidRecurrence("khoảng lặp phải từ 1 đến %d", maxRecurrenceGap)
	}
	if opt.Count < 0 {
		return nil, invalidRecurrence("COUNT không hợp lệ")
	}
	if len(opt.Byhour) > 0 || len(opt.Byminute) > 0 || len(opt.Bysecond) > 0 || len(opt.Bysetpos) > 0 ||
		len(opt.Byweekno) > 0 || len(opt.Byyearday) > 0 || len(opt.Byeaster) > 0 {
		return nil, invalidRecurrence("quy tắc lặp dùng thành phần chưa hỗ trợ")
	}
	opt.Dtstart = time.Time{}
	return opt, nil
}

type occurrence struct {
	due  time.Time
	rule string
}

func nextOccurrence(rule, tz string, due, completedAt time.Time) (occurrence, bool, error) {
	opt, err := parseRecurrence(rule)
	if err != nil {
		return occurrence{}, false, err
	}
	loc := recurrenceLocation(tz)
	if opt.Count == 1 {
		return occurrence{}, false, nil
	}

	nextRule := *opt
	if opt.Count > 1 {
		nextRule.Count = opt.Count - 1
	}

	opt.Count = 0
	opt.Dtstart = due.In(loc)
	r, err := rrule.NewRRule(*opt)
	if err != nil {
		return occurrence{}, false, invalidRecurrence("quy tắc lặp không hợp lệ: %v", err)
	}

	after := due
	if completedAt.After(after) {
		after = completedAt
	}
	next := r.After(after.In(loc), false)
	if next.IsZero() {
		return occurrence{}, false, nil
	}
	return occurrence{due: next.UTC(), rule: nextRule.RRuleString()}, true, nil
}

func recurrenceLocation(tz string) *time.Location {
	for _, name := range []string{tz, defaultRecurrenceTZ} {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	return time.UTC
}
