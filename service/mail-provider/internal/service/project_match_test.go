package service

import (
	"testing"

	"taskmanager/service/mail-provider/internal/model"
)

func TestMatchProjectByRulesPicksHighestOverlap(t *testing.T) {
	projects := []model.ProjectCandidate{
		{ID: "other", Name: "Marketing nội bộ", Description: "Chiến dịch quảng cáo quý"},
		{ID: "acme", Name: "Website khách hàng ACME", Description: "Thiết kế website cho ACME"},
	}
	id, score := matchProjectByRules("Cập nhật website ACME cho khách hàng trước thứ sáu", projects)
	if id != "acme" {
		t.Fatalf("phải chọn acme, nhận %q", id)
	}
	if score < 0.6 {
		t.Fatalf("score phải >= 0.6, nhận %v", score)
	}
}

func TestMatchProjectByRulesZeroWhenNoOverlap(t *testing.T) {
	projects := []model.ProjectCandidate{{ID: "acme", Name: "Website khách hàng ACME", Description: "Thiết kế website"}}
	id, score := matchProjectByRules("Khuyến mãi cuối năm giảm giá sâu", projects)
	if id != "" || score != 0 {
		t.Fatalf("không trùng token phải trả rỗng, nhận %q %v", id, score)
	}
}

func TestResolveProjectRejectsUnknownID(t *testing.T) {
	projects := []model.ProjectCandidate{{ID: "pr1", Name: "A"}}
	_, ok := resolveProject(model.MailDraft{ProjectID: "ghost", ProjectConfidence: 0.9}, projects, 0.6)
	if ok {
		t.Fatal("id không có trong danh sách phải bị từ chối")
	}
}

func TestResolveProjectRejectsBelowThreshold(t *testing.T) {
	projects := []model.ProjectCandidate{{ID: "pr1", Name: "A"}}
	if _, ok := resolveProject(model.MailDraft{ProjectID: "pr1", ProjectConfidence: 0.59}, projects, 0.6); ok {
		t.Fatal("0.59 < 0.6 phải bị từ chối")
	}
	project, ok := resolveProject(model.MailDraft{ProjectID: "pr1", ProjectConfidence: 0.6}, projects, 0.6)
	if !ok || project.ID != "pr1" {
		t.Fatalf("0.6 phải được chấp nhận, nhận %+v %v", project, ok)
	}
}
