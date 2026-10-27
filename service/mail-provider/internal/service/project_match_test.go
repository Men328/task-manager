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
	id, score := matchProjectByRules("Cập nhật thiết kế website ACME cho khách hàng trước thứ sáu", projects)
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

func TestMatchProjectByRulesExactScore(t *testing.T) {
	// token project: website, acme, thiết, kế ("cho" là stopword) → mẫu số 4.
	projects := []model.ProjectCandidate{{ID: "acme", Name: "Website ACME", Description: "Thiết kế website cho ACME"}}
	id, score := matchProjectByRules("Cập nhật website ACME", projects)
	if id != "acme" || score != 0.5 {
		t.Fatalf("trùng 2/4 token phải ra 0.5, nhận %q %v", id, score)
	}
}

func TestMatchProjectByRulesSingleTokenOverlapIsZero(t *testing.T) {
	projects := []model.ProjectCandidate{{ID: "home", Name: "Nhà"}}
	id, score := matchProjectByRules("Cuối tuần này sửa nhà giúp mình nhé", projects)
	if id != "" || score != 0 {
		t.Fatalf("chỉ trùng 1 token phải ra 0, nhận %q %v", id, score)
	}
}

func TestMatchProjectByRulesShopPromoBelowThreshold(t *testing.T) {
	projects := []model.ProjectCandidate{{ID: "acme", Name: "Khách hàng ACME", Description: "website bán hàng"}}
	_, score := matchProjectByRules("Cửa hàng ABC giảm giá 50%, quý khách mua hàng ngay hôm nay, giá bán ưu đãi", projects)
	if score >= 0.6 {
		t.Fatalf("email quảng cáo shop phải dưới 0.6, nhận %v", score)
	}
}

func TestMatchProjectByRulesTieGoesToEarlierProject(t *testing.T) {
	projects := []model.ProjectCandidate{
		{ID: "first", Name: "Alpha Beta"},
		{ID: "second", Name: "Beta Alpha"},
	}
	id, score := matchProjectByRules("alpha beta gamma", projects)
	if id != "first" {
		t.Fatalf("hoà điểm phải chọn project đứng trước, nhận %q", id)
	}
	if score < 0.66 || score > 0.67 {
		t.Fatalf("2 token với mẫu số tối thiểu 3 phải ra ~0.667, nhận %v", score)
	}
}

func TestMatchProjectByRulesCapsDenominatorAtFive(t *testing.T) {
	projects := []model.ProjectCandidate{{
		ID:          "big",
		Name:        "Hội nghị khoa học quốc tế",
		Description: "diễn giả poster phiên toàn thể",
	}}
	id, score := matchProjectByRules("Gửi poster cho hội nghị", projects)
	if id != "big" || score != 0.6 {
		t.Fatalf("trùng 3 token (project có 12 token), mẫu số cap 5 phải ra 0.6, nhận %q %v", id, score)
	}
}

func TestProjectTokensDropsVietnameseStopwords(t *testing.T) {
	tokens := projectTokens("Thông báo ngày họp với khách hàng")
	if len(tokens) != 1 {
		t.Fatalf("chỉ còn token \"họp\", nhận %v", tokens)
	}
	if _, ok := tokens["họp"]; !ok {
		t.Fatalf("thiếu token \"họp\", nhận %v", tokens)
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
