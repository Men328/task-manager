package service

import (
	"strings"
	"unicode"

	"taskmanager/service/mail-provider/internal/model"
)

const (
	reasonNoProjectMatched = "no_project_matched"
	maxProjectTokens       = 5
	// Mẫu số tối thiểu: project tên ngắn (1-2 token) không được đạt điểm tuyệt đối dễ dàng.
	minProjectDenominator = 3
	// Số token trùng tối thiểu để có điểm khác 0, tránh khớp nhầm vì một âm tiết chung.
	minProjectOverlap = 2
)

// projectStopwords gồm hư từ và các âm tiết tiếng Việt rất phổ biến trong email
// (chào hỏi, thông báo, quảng cáo) — để chúng không kéo email lạ vào project thật.
var projectStopwords = map[string]struct{}{
	"và": {}, "của": {}, "cho": {}, "các": {}, "những": {}, "là": {}, "có": {}, "được": {},
	"hàng": {}, "khách": {}, "công": {}, "việc": {}, "bạn": {}, "anh": {}, "chị": {}, "em": {},
	"quý": {}, "xin": {}, "chào": {}, "cảm": {}, "ơn": {}, "thông": {}, "báo": {}, "ngày": {},
	"giờ": {}, "tại": {}, "với": {}, "này": {}, "đã": {}, "sẽ": {}, "không": {}, "một": {},
	"người": {}, "từ": {}, "về": {}, "theo": {}, "như": {}, "khi": {}, "để": {}, "trong": {},
	"trên": {}, "mới": {},
	"the": {}, "and": {}, "for": {}, "with": {}, "to": {}, "of": {}, "a": {}, "an": {}, "in": {}, "on": {},
}

func projectTokens(s string) map[string]struct{} {
	out := map[string]struct{}{}
	fields := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
	for _, token := range fields {
		if len([]rune(token)) < 2 {
			continue
		}
		if _, stop := projectStopwords[token]; stop {
			continue
		}
		out[token] = struct{}{}
	}
	return out
}

// matchProjectByRules chấm điểm từng project theo số token trùng với nội dung email
// và trả về project điểm cao nhất; hoà điểm thì chọn project đứng trước trong danh sách.
// Điểm = số token trùng / max(min(số token project, 5), 3), chỉ tính khi trùng ít nhất 2 token.
func matchProjectByRules(text string, projects []model.ProjectCandidate) (string, float64) {
	emailTokens := projectTokens(text)
	bestID := ""
	best := 0.0
	for _, project := range projects {
		tokens := projectTokens(project.Name + " " + project.Description)
		if len(tokens) == 0 {
			continue
		}
		overlap := 0
		for token := range tokens {
			if _, ok := emailTokens[token]; ok {
				overlap++
			}
		}
		if overlap < minProjectOverlap {
			continue
		}
		denom := len(tokens)
		if denom > maxProjectTokens {
			denom = maxProjectTokens
		}
		if denom < minProjectDenominator {
			denom = minProjectDenominator
		}
		score := float64(overlap) / float64(denom)
		if score > 1 {
			score = 1
		}
		if score > best {
			best, bestID = score, project.ID
		}
	}
	return bestID, best
}

// resolveProject chỉ chấp nhận project của draft khi project đó có trong danh sách
// và độ tin cậy đạt ngưỡng.
func resolveProject(draft model.MailDraft, projects []model.ProjectCandidate, threshold float64) (model.ProjectCandidate, bool) {
	if draft.ProjectID == "" || draft.ProjectConfidence < threshold {
		return model.ProjectCandidate{}, false
	}
	for _, project := range projects {
		if project.ID == draft.ProjectID {
			return project, true
		}
	}
	return model.ProjectCandidate{}, false
}
