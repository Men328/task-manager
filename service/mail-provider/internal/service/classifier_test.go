package service

import (
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

func TestClassifyByRulesDetectsTask(t *testing.T) {
	draft := classifyByRules(model.EmailMessage{
		Subject: "Cần nộp báo cáo tháng 10",
		Body:    "Vui lòng hoàn thành trước hạn chót.",
	})
	if draft.Category != model.CategoryTask {
		t.Fatalf("phải phân loại là task, nhận %q", draft.Category)
	}
	if !draft.Actionable {
		t.Fatal("task từ rule phải actionable")
	}
	if draft.Title != "Cần nộp báo cáo tháng 10" {
		t.Fatalf("title phải lấy từ subject, nhận %q", draft.Title)
	}
}

func TestClassifyByRulesDetectsScheduleWithTime(t *testing.T) {
	draft := classifyByRules(model.EmailMessage{
		Subject: "Nhắc lịch họp nhóm",
		Body:    "Cuộc họp diễn ra 2026-10-05 lúc 09:00.",
	})
	if draft.Category != model.CategorySchedule {
		t.Fatalf("phải phân loại là schedule, nhận %q", draft.Category)
	}
	if draft.StartAt == nil {
		t.Fatal("rule phải bóc được start_at")
	}
	if draft.StartAt.Year() != 2026 || draft.StartAt.Month() != time.October || draft.StartAt.Day() != 5 {
		t.Fatalf("start_at sai: %v", draft.StartAt)
	}
}

func TestClassifyByRulesDetectsEvent(t *testing.T) {
	draft := classifyByRules(model.EmailMessage{
		Subject: "Thư mời tham dự hội thảo công nghệ",
		Body:    "Sự kiện sẽ diễn ra tại hội trường.",
	})
	if draft.Category != model.CategoryEvent {
		t.Fatalf("phải phân loại là event, nhận %q", draft.Category)
	}
}

func TestClassifyByRulesFallsBackToOther(t *testing.T) {
	draft := classifyByRules(model.EmailMessage{
		Subject: "Bản tin nội bộ số 12",
		Body:    "Tổng hợp tin tức trong tuần.",
	})
	if draft.Category != model.CategoryOther {
		t.Fatalf("email không khớp rule phải là other, nhận %q", draft.Category)
	}
	if draft.Actionable {
		t.Fatal("other phải không actionable")
	}
	if draft.Reason != "no_rule_matched" {
		t.Fatalf("reason phải là no_rule_matched, nhận %q", draft.Reason)
	}
}

func TestParseFirstTimeSupportsBothFormats(t *testing.T) {
	iso := parseFirstTime("họp lúc 2026-10-05 09:30")
	if iso == nil || iso.Hour() != 9 || iso.Minute() != 30 {
		t.Fatalf("không bóc được giờ ISO: %v", iso)
	}

	dayFirst := parseFirstTime("diễn ra 12/11/2026 14:00")
	if dayFirst == nil || dayFirst.Day() != 12 || dayFirst.Month() != time.November || dayFirst.Hour() != 14 {
		t.Fatalf("không bóc được ngày dạng dd/mm/yyyy: %v", dayFirst)
	}

	if parseFirstTime("không có ngày nào") != nil {
		t.Fatal("chuỗi không có ngày phải trả nil")
	}
}

func TestClassifyByRulesKeepsClassifierStableForInvitations(t *testing.T) {
	draft := classifyByRules(model.EmailMessage{
		Subject: "Thiệp mời sinh nhật",
		Body:    "Mời bạn tham gia tiệc sinh nhật.",
	})
	if draft.Category != model.CategoryEvent {
		t.Fatalf("thiệp mời phải là event, nhận %q", draft.Category)
	}
	if draft.Actionable != true {
		t.Fatal("thiệp mời phải actionable")
	}
}
