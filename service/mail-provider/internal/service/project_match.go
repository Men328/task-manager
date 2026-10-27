package service

import (
	"strings"
	"unicode"

	"taskmanager/service/mail-provider/internal/model"
)

const (
	reasonNoProjectMatched = "no_project_matched"
	maxProjectTokens       = 5
)

var projectStopwords = map[string]struct{}{
	"và": {}, "của": {}, "cho": {}, "các": {}, "những": {}, "là": {}, "có": {}, "được": {},
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

// matchProjectByRules scores each project by token overlap with the email text
// and returns the best one; ties go to the earlier project in the list.
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
		denom := len(tokens)
		if denom > maxProjectTokens {
			denom = maxProjectTokens
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

// resolveProject accepts the draft's project only when it exists in the list
// and its confidence reaches the threshold.
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
