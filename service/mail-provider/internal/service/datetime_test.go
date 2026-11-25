package service

import (
	"testing"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

func testLocation() *time.Location {
	return time.FixedZone("ICT", 7*60*60)
}

// Thứ ba 24/11/2026 22:59 giờ Việt Nam.
func testReceivedAt() time.Time {
	return time.Date(2026, 11, 24, 22, 59, 0, 0, testLocation())
}

func assertLocalTime(t *testing.T, got *time.Time, want time.Time) {
	t.Helper()
	if got == nil {
		t.Fatalf("phải bóc được thời gian %v, nhận nil", want)
	}
	if !got.Equal(want) {
		t.Fatalf("thời gian sai: muốn %v, nhận %v", want, got.In(testLocation()))
	}
}

func TestExtractScheduleKickoffEmail(t *testing.T) {
	text := "Họp kickoff dự án CRM\nMen Tran Thi mời tham dự buổi họp kickoff dự án CRM lúc 9:00 sáng mai tại phòng họp tầng 3, kéo dài khoảng một tiếng. Cần sắp xếp thời gian tham dự."
	got := extractSchedule(text, testReceivedAt(), testLocation())

	assertLocalTime(t, got.Start, time.Date(2026, 11, 25, 9, 0, 0, 0, testLocation()))
	assertLocalTime(t, got.End, time.Date(2026, 11, 25, 10, 0, 0, 0, testLocation()))
	if got.AllDay {
		t.Fatal("có giờ cụ thể thì không phải cả ngày")
	}
	if place := extractLocation(text); place != "phòng họp tầng 3" {
		t.Fatalf("địa điểm sai: %q", place)
	}
}

func TestExtractScheduleFormats(t *testing.T) {
	loc := testLocation()
	cases := []struct {
		name string
		text string
		want time.Time
	}{
		{"iso", "họp lúc 2026-10-05 09:30", time.Date(2026, 10, 5, 9, 30, 0, 0, loc)},
		{"dd/mm/yyyy", "diễn ra 12/11/2026 14:00", time.Date(2026, 11, 12, 14, 0, 0, 0, loc)},
		{"dd/mm không năm, giờ chiều", "Hẹn 3h chiều ngày 28/11 nhé", time.Date(2026, 11, 28, 15, 0, 0, 0, loc)},
		{"giờ trước ngày", "Lúc 14h30 ngày 30/11/2026 tại hội trường A2", time.Date(2026, 11, 30, 14, 30, 0, 0, loc)},
		{"ngày tháng bằng chữ", "vào 8 giờ ngày 2 tháng 12", time.Date(2026, 12, 2, 8, 0, 0, 0, loc)},
		{"hôm nay", "Chiều nay lúc 4 giờ họp nhanh", time.Date(2026, 11, 24, 16, 0, 0, 0, loc)},
		{"ngày kia", "ngày kia 10:15 gặp khách", time.Date(2026, 11, 26, 10, 15, 0, 0, loc)},
		{"thứ sáu", "Hẹn thứ 6 lúc 9h", time.Date(2026, 11, 27, 9, 0, 0, 0, loc)},
		{"thứ hai tuần sau", "Họp vào thứ hai tuần sau 8:00", time.Date(2026, 11, 30, 8, 0, 0, 0, loc)},
		{"pm", "Call at 2pm tomorrow", time.Date(2026, 11, 25, 14, 0, 0, 0, loc)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := extractSchedule(tc.text, testReceivedAt(), loc)
			assertLocalTime(t, got.Start, tc.want)
		})
	}
}

func TestExtractScheduleDateOnlyIsAllDay(t *testing.T) {
	got := extractSchedule("Hội thảo diễn ra ngày 05/12/2026", testReceivedAt(), testLocation())
	assertLocalTime(t, got.Start, time.Date(2026, 12, 5, 0, 0, 0, 0, testLocation()))
	if !got.AllDay {
		t.Fatal("chỉ có ngày thì phải là cả ngày")
	}
}

func TestExtractScheduleTimeRange(t *testing.T) {
	got := extractSchedule("Workshop 25/11/2026 từ 9:00 đến 11:30", testReceivedAt(), testLocation())
	assertLocalTime(t, got.Start, time.Date(2026, 11, 25, 9, 0, 0, 0, testLocation()))
	assertLocalTime(t, got.End, time.Date(2026, 11, 25, 11, 30, 0, 0, testLocation()))
}

func TestExtractScheduleIgnoresNonTimes(t *testing.T) {
	for _, text := range []string{"không có ngày nào", "Lớp có 3 học sinh mới", "phiên bản 2.5 đã ra mắt"} {
		if got := extractSchedule(text, testReceivedAt(), testLocation()); got.Start != nil {
			t.Fatalf("%q không có thời gian, nhận %v", text, got.Start)
		}
	}
}

func TestExtractLocation(t *testing.T) {
	cases := map[string]string{
		"Địa điểm: Hội trường A2, toà nhà B":         "Hội trường A2",
		"Buổi họp tổ chức tại Phòng 301 lúc 9h":      "Phòng 301",
		"Họp online qua Google Meet":                 "",
		"Hiện tại chưa có lịch cụ thể":               "",
		"Gặp nhau ở quán cafe Cộng, 15 Hàng Bài nhé": "quán cafe Cộng",
		"Location: Zoom": "Zoom",
		"Tiệc sinh nhật tại nhà hàng Sen Tây Hồ vào tối thứ 7": "nhà hàng Sen Tây Hồ",
	}
	for text, want := range cases {
		if got := extractLocation(text); got != want {
			t.Fatalf("extractLocation(%q) = %q, muốn %q", text, got, want)
		}
	}
}

func TestFillScheduleKeepsAnalyzerValues(t *testing.T) {
	start := time.Date(2026, 11, 25, 2, 0, 0, 0, time.UTC)
	draft := model.MailDraft{Category: model.CategorySchedule, StartAt: &start, Location: "Phòng A"}
	fillScheduleFromText(&draft, model.EmailMessage{
		Body:       "Họp lúc 15:00 ngày 30/11 tại phòng họp tầng 3, kéo dài 2 tiếng",
		ReceivedAt: testReceivedAt(),
	}, testLocation())

	if !draft.StartAt.Equal(start) || draft.Location != "Phòng A" {
		t.Fatalf("không được ghi đè giá trị DeepSeek đã trả: %+v", draft)
	}
	assertLocalTime(t, draft.EndAt, start.Add(2*time.Hour))
}

func TestFillScheduleFillsMissingValues(t *testing.T) {
	draft := model.MailDraft{Category: model.CategoryEvent}
	fillScheduleFromText(&draft, model.EmailMessage{
		Subject:    "Mời dự hội thảo",
		Body:       "Hội thảo lúc 8h30 sáng mai tại Hội trường lớn tầng 2.",
		ReceivedAt: testReceivedAt(),
	}, testLocation())

	assertLocalTime(t, draft.StartAt, time.Date(2026, 11, 25, 8, 30, 0, 0, testLocation()))
	if draft.Location != "Hội trường lớn tầng 2" {
		t.Fatalf("địa điểm sai: %q", draft.Location)
	}
}
