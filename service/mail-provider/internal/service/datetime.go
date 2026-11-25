package service

import (
	"regexp"
	"strings"
	"time"
)

// Bóc ngày giờ / thời lượng / địa điểm từ nội dung email bằng rule (tiếng Việt + một ít tiếng Anh).
// Mọi mốc tương đối ("sáng mai", "thứ 6 tuần sau") được tính theo thời điểm nhận email
// trong múi giờ của người dùng, để "9:00 sáng mai" ra đúng 9:00 giờ Việt Nam.

// mailSchedule là kết quả bóc thời gian của một email.
type mailSchedule struct {
	Start  *time.Time
	End    *time.Time
	AllDay bool
}

var (
	isoDatePattern    = regexp.MustCompile(`\b(\d{4})-(\d{1,2})-(\d{1,2})\b`)
	slashDatePattern  = regexp.MustCompile(`\b(\d{1,2})([/.-])(\d{1,2})(?:[/.-](\d{2,4}))?\b`)
	wordDatePattern   = regexp.MustCompile(`(?i)ngày\s+(\d{1,2})\s+tháng\s+(\d{1,2})(?:\s+năm\s+(\d{4}))?`)
	clockPattern      = regexp.MustCompile(`(?i)\b(\d{1,2})(?::|h|g|\s*giờ)(?:\s*(\d{2}))?(?:\s*phút)?(?:\s*(am|pm))?`)
	ampmPattern       = regexp.MustCompile(`(?i)\b(\d{1,2})\s*(am|pm)\b`)
	periodPattern     = regexp.MustCompile(`(?i)(sáng|trưa|chiều|tối|đêm)`)
	weekdayPattern    = regexp.MustCompile(`(?i)(thứ\s*(?:hai|ba|tư|năm|sáu|bảy|bẩy|[2-7])|chủ\s*nhật)(\s+tuần\s+(?:sau|tới))?`)
	durationPattern   = regexp.MustCompile(`(?i)(?:kéo\s+dài|trong|duration|lasting|for)\s+(?:khoảng\s+|tầm\s+|chừng\s+|about\s+)?(\d+(?:[.,]\d+)?|một|hai|ba|bốn|năm|nửa)\s*(tiếng|giờ|h|phút|hours?|minutes?|mins?)(\s+rưỡi)?`)
	rangeEndPattern   = regexp.MustCompile(`(?i)(?:-|–|đến|tới|to|until)\s*(\d{1,2})\s*(?::|h|g|giờ)\s*(\d{2})?`)
	locationPattern   = regexp.MustCompile(`(?i)(?:^|[\s,(])(?:địa\s+điểm|location|venue|where)\s*[:：-]\s*([^\n]+)`)
	atPlacePattern    = regexp.MustCompile(`(?i)(?:^|[\s,(])(?:tại|ở|at)\s+([^\n]+)`)
	placeStopPattern  = regexp.MustCompile(`(?i)[.;!?\n]|,\s|\s(?:lúc|vào|từ|kéo\s+dài|trong\s+khoảng|để|nhằm|với|cùng|cần|vui\s+lòng|xin|mong|at\s+\d|on)(?:\s|$)`)
	placeHintPattern  = regexp.MustCompile(`(?i)phòng|tầng|hội\s+trường|toà|tòa|nhà|văn\s+phòng|cơ\s+sở|khách\s+sạn|nhà\s+hàng|quán|cafe|café|số\s+\d|đường|phố|quận|huyện|phường|tp\.?|thành\s+phố|hà\s+nội|hồ\s+chí\s+minh|đà\s+nẵng|google\s+meet|zoom|teams|online|trực\s+tuyến|room|floor|hall|building|street|office`)
	vietnameseNumbers = map[string]float64{"một": 1, "hai": 2, "ba": 3, "bốn": 4, "năm": 5, "nửa": 0.5, "rưỡi": 0.5}
	vietnameseWeekday = map[string]time.Weekday{
		"hai": time.Monday, "2": time.Monday, "ba": time.Tuesday, "3": time.Tuesday,
		"tư": time.Wednesday, "4": time.Wednesday, "năm": time.Thursday, "5": time.Thursday,
		"sáu": time.Friday, "6": time.Friday, "bảy": time.Saturday, "bẩy": time.Saturday, "7": time.Saturday,
	}
)

// extractSchedule bóc thời gian bắt đầu/kết thúc của email. ref là lúc nhận email; loc là múi giờ người dùng.
func extractSchedule(text string, ref time.Time, loc *time.Location) mailSchedule {
	if loc == nil {
		loc = defaultLocation()
	}
	if ref.IsZero() {
		ref = time.Now()
	}
	ref = ref.In(loc)
	lower := strings.ToLower(text)

	year, month, day, hasDate := findDate(lower, ref)
	hour, minute, clockEnd, hasClock := findClock(lower)

	if !hasDate && !hasClock {
		return mailSchedule{}
	}
	if !hasDate {
		year, month, day = ref.Year(), int(ref.Month()), ref.Day()
	}
	if !validDate(year, month, day) {
		return mailSchedule{}
	}

	if !hasClock {
		start := time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
		return mailSchedule{Start: &start, AllDay: true}
	}

	start := time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc)
	out := mailSchedule{Start: &start}

	if end := rangeEnd(lower[clockEnd:], start); end != nil {
		out.End = end
	} else if d := findDuration(lower); d > 0 {
		end := start.Add(d)
		out.End = &end
	}
	return out
}

func findDate(lower string, ref time.Time) (int, int, int, bool) {
	if m := isoDatePattern.FindStringSubmatch(lower); m != nil {
		return atoi(m[1]), atoi(m[2]), atoi(m[3]), true
	}
	if m := wordDatePattern.FindStringSubmatch(lower); m != nil {
		year := ref.Year()
		if m[3] != "" {
			year = atoi(m[3])
		}
		return year, atoi(m[2]), atoi(m[1]), true
	}
	for _, m := range slashDatePattern.FindAllStringSubmatch(lower, -1) {
		day, month := atoi(m[1]), atoi(m[3])
		if day < 1 || day > 31 || month < 1 || month > 12 {
			continue
		}
		// "2.5" (phiên bản) hay "2-3 người" không phải ngày: dấu "." / "-" chỉ nhận khi có năm.
		if m[2] != "/" && m[4] == "" {
			continue
		}
		year := ref.Year()
		if m[4] != "" {
			year = atoi(m[4])
			if year < 100 {
				year += 2000
			}
		}
		return year, month, day, true
	}

	relative := []struct {
		words []string
		days  int
	}{
		{[]string{"ngày kia", "ngày mốt", "hôm kia", "day after tomorrow"}, 2},
		{[]string{"ngày mai", "sáng mai", "trưa mai", "chiều mai", "tối mai", "đêm mai", "mai lúc", "tomorrow"}, 1},
		{[]string{"hôm nay", "sáng nay", "trưa nay", "chiều nay", "tối nay", "today", "tonight"}, 0},
	}
	for _, group := range relative {
		for _, word := range group.words {
			if strings.Contains(lower, word) {
				d := ref.AddDate(0, 0, group.days)
				return d.Year(), int(d.Month()), d.Day(), true
			}
		}
	}

	if m := weekdayPattern.FindStringSubmatch(lower); m != nil {
		target, ok := parseWeekday(m[1])
		if ok {
			offset := (int(target) - int(ref.Weekday()) + 7) % 7
			if strings.TrimSpace(m[2]) != "" {
				// "thứ 6 tuần sau": ngày đó trong tuần kế tiếp (tuần bắt đầu thứ hai).
				mondayOffset := (int(time.Monday) - int(ref.Weekday()) + 7) % 7
				if mondayOffset == 0 {
					mondayOffset = 7
				}
				offset = mondayOffset + (int(target)-int(time.Monday)+7)%7
			}
			d := ref.AddDate(0, 0, offset)
			return d.Year(), int(d.Month()), d.Day(), true
		}
	}
	return 0, 0, 0, false
}

func parseWeekday(raw string) (time.Weekday, bool) {
	normalized := strings.Join(strings.Fields(raw), " ")
	if normalized == "chủ nhật" || normalized == "cn" {
		return time.Sunday, true
	}
	key := strings.TrimSpace(strings.TrimPrefix(normalized, "thứ"))
	weekday, ok := vietnameseWeekday[key]
	return weekday, ok
}

// findClock tìm giờ đầu tiên trong email ("9:00", "9h30", "14 giờ", "9 giờ sáng", "2pm")
// và trả vị trí kết thúc của cụm giờ để tìm giờ kết thúc ngay sau đó.
func findClock(lower string) (int, int, int, bool) {
	type hit struct{ hour, minute, start, end int }
	var best *hit
	consider := func(h hit) {
		if best == nil || h.start < best.start {
			best = &h
		}
	}

	for _, idx := range clockPattern.FindAllStringSubmatchIndex(lower, -1) {
		// "12/11", "2026-10" hay "3 học sinh" không phải giờ.
		if idx[2] > 0 && isDateSeparatorOrDigit(lower[idx[2]-1]) {
			continue
		}
		if idx[1] < len(lower) && isASCIILetter(lower[idx[1]]) {
			continue
		}
		hour := atoi(lower[idx[2]:idx[3]])
		minute := 0
		if idx[4] >= 0 {
			minute = atoi(lower[idx[4]:idx[5]])
		}
		if !validClock(hour, minute) {
			continue
		}
		suffix := ""
		if idx[6] >= 0 {
			suffix = lower[idx[6]:idx[7]]
		}
		consider(hit{adjustHour(hour, suffix, periodNear(lower, idx[0], idx[1])), minute, idx[0], idx[1]})
		break
	}
	for _, idx := range ampmPattern.FindAllStringSubmatchIndex(lower, -1) {
		if idx[2] > 0 && isDateSeparatorOrDigit(lower[idx[2]-1]) {
			continue
		}
		hour := atoi(lower[idx[2]:idx[3]])
		if hour < 1 || hour > 12 {
			continue
		}
		consider(hit{adjustHour(hour, lower[idx[4]:idx[5]], ""), 0, idx[0], idx[1]})
		break
	}
	if best == nil {
		return 0, 0, 0, false
	}
	return best.hour, best.minute, best.end, true
}

func isDateSeparatorOrDigit(b byte) bool {
	return b == '/' || b == '-' || b == '.' || (b >= '0' && b <= '9')
}

func isASCIILetter(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// periodNear tìm "sáng/chiều/tối" ngay quanh cụm giờ ("9 giờ sáng", "chiều nay lúc 3h").
func periodNear(lower string, start int, end int) string {
	after := lower[end:]
	if len(after) > 24 {
		after = after[:24]
	}
	if m := periodPattern.FindStringSubmatch(after); m != nil {
		return m[1]
	}
	from := start - 32
	if from < 0 {
		from = 0
	}
	before := periodPattern.FindAllStringSubmatch(lower[from:start], -1)
	if len(before) > 0 {
		return before[len(before)-1][1]
	}
	return ""
}

func adjustHour(hour int, suffix string, period string) int {
	switch suffix {
	case "pm":
		if hour < 12 {
			return hour + 12
		}
		return hour
	case "am":
		if hour == 12 {
			return 0
		}
		return hour
	}
	switch period {
	case "chiều", "tối":
		if hour < 12 {
			return hour + 12
		}
	case "trưa":
		if hour < 11 {
			return hour + 12
		}
	case "đêm":
		if hour >= 6 && hour < 12 {
			return hour + 12
		}
	}
	return hour
}

func rangeEnd(after string, start time.Time) *time.Time {
	if len(after) > 24 {
		after = after[:24]
	}
	m := rangeEndPattern.FindStringSubmatch(after)
	if m == nil {
		return nil
	}
	hour, minute := atoi(m[1]), atoi(m[2])
	if !validClock(hour, minute) {
		return nil
	}
	if hour < start.Hour() && hour < 12 {
		hour += 12
	}
	end := time.Date(start.Year(), start.Month(), start.Day(), hour, minute, 0, 0, start.Location())
	if !end.After(start) {
		return nil
	}
	return &end
}

func findDuration(lower string) time.Duration {
	idx := durationPattern.FindStringSubmatchIndex(lower)
	if idx == nil || (idx[1] < len(lower) && isASCIILetter(lower[idx[1]])) {
		return 0
	}
	m := make([]string, len(idx)/2)
	for i := range m {
		if idx[2*i] >= 0 {
			m[i] = lower[idx[2*i]:idx[2*i+1]]
		}
	}
	amount, ok := vietnameseNumbers[m[1]]
	if !ok {
		amount = parseFloat(strings.ReplaceAll(m[1], ",", "."))
	}
	if strings.TrimSpace(m[3]) != "" {
		amount += 0.5
	}
	if amount <= 0 {
		return 0
	}
	unit := time.Hour
	switch m[2] {
	case "phút", "p", "minute", "minutes", "min", "mins":
		unit = time.Minute
	}
	d := time.Duration(amount * float64(unit))
	if d > 24*time.Hour {
		return 0
	}
	return d
}

// extractLocation lấy địa điểm từ "Địa điểm: ..." hoặc cụm "tại <nơi chốn>".
func extractLocation(text string) string {
	if m := locationPattern.FindStringSubmatch(text); m != nil {
		if place := cleanPlace(m[1]); place != "" {
			return place
		}
	}
	for _, m := range atPlacePattern.FindAllStringSubmatchIndex(text, -1) {
		// Bỏ "hiện tại", "tại sao", "ở đây"...
		prefix := strings.ToLower(text[max(0, m[0]-6):m[2]])
		if strings.Contains(prefix, "hiện") {
			continue
		}
		place := cleanPlace(text[m[2]:m[3]])
		lowerPlace := strings.ToLower(place)
		if place == "" || strings.HasPrefix(lowerPlace, "sao") || strings.HasPrefix(lowerPlace, "đây") {
			continue
		}
		if !placeHintPattern.MatchString(place) {
			continue
		}
		return place
	}
	return ""
}

func cleanPlace(raw string) string {
	place := raw
	if loc := placeStopPattern.FindStringIndex(place); loc != nil {
		place = place[:loc[0]]
	}
	place = strings.Trim(strings.TrimSpace(place), " ,.;:-")
	return truncateRunes(place, 200)
}

func defaultLocation() *time.Location {
	if loc, err := time.LoadLocation("Asia/Ho_Chi_Minh"); err == nil {
		return loc
	}
	return time.FixedZone("ICT", 7*60*60)
}

func parseFloat(value string) float64 {
	whole, frac, _ := strings.Cut(value, ".")
	out := float64(atoi(whole))
	if frac != "" {
		scale := 1.0
		for range frac {
			scale *= 10
		}
		out += float64(atoi(frac)) / scale
	}
	return out
}
