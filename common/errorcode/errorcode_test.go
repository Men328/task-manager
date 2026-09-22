package errorcode

import (
	"strings"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
)

func declaredCodes() []string {
	return []string{
		CommonInvalidArgument,
		CommonNotFound,
		CommonAlreadyExists,
		CommonFailedPrecondition,
		CommonInternal,
		CommonUnavailable,
		IdentityEmailRequired,
		IdentityDisplayNameRequired,
		IdentityProfileIDRequired,
		IdentityProfileNotFound,
		TaskProfileIDRequired,
		TaskWorkspaceIDRequired,
		TaskTitleRequired,
		TaskIDRequired,
		TaskIDAndStatusIDRequired,
		TaskNotFound,
		TaskParentNotInProfile,
		TaskHasSubtasks,
		TaskStatusSame,
		TaskStatusNotInProfile,
		TaskStatusProfileMismatch,
		TaskNoStatusAvailable,
		TaskTransitionNotAllowed,
		TaskCycleDetected,
		TaskStatusNameRequired,
		TaskStatusSlugRequired,
		TaskStatusSlugAlreadyExists,
		TaskStatusIDRequired,
		TaskStatusNotFound,
		TaskStatusInUse,
		TaskTransitionFieldsRequired,
		TaskTransitionSameStatus,
		TaskTransitionIDRequired,
		TaskTransitionNotFound,
		TaskTransitionAlreadyExists,
	}
}

func TestDeclaredCodesExistInCatalog(t *testing.T) {
	for _, code := range declaredCodes() {
		if _, ok := Lookup(code); !ok {
			t.Errorf("mã %q chưa có trong error_codes.json", code)
		}
	}
}

func TestCatalogEntriesAreUsable(t *testing.T) {
	for code, entry := range Catalog() {
		if strings.TrimSpace(entry.Owner) == "" {
			t.Errorf("%s: thiếu owner", code)
		}
		if strings.TrimSpace(entry.Message.VI) == "" || strings.TrimSpace(entry.Message.EN) == "" {
			t.Errorf("%s: thiếu message vi/en", code)
		}
		if entry.Owner == "client" {
			continue
		}
		if _, ok := grpcCodeNames[entry.GRPCCode]; !ok {
			t.Errorf("%s: grpcCode %q không hợp lệ", code, entry.GRPCCode)
		}
	}
}

func TestMessageFallsBackAcrossLanguages(t *testing.T) {
	vi, ok := Message(TaskTitleRequired, "vi")
	if !ok || vi == "" {
		t.Fatalf("không lấy được message tiếng Việt")
	}
	en, ok := Message(TaskTitleRequired, "en-US")
	if !ok || en == "" {
		t.Fatalf("không lấy được message tiếng Anh")
	}
	if vi == en {
		t.Fatalf("message vi và en không được trùng nhau")
	}
	if _, ok := Message("KHONG_TON_TAI", "vi"); ok {
		t.Fatalf("mã không tồn tại phải trả ok=false")
	}
}

func TestDefaultLanguageIsEnglish(t *testing.T) {
	en, ok := Message(TaskTitleRequired, "en")
	if !ok || en == "" {
		t.Fatalf("không lấy được message tiếng Anh")
	}
	if got := DefaultMessage(TaskTitleRequired); got != en {
		t.Fatalf("DefaultMessage = %q, muốn %q", got, en)
	}
	if got := DefaultMessage("KHONG_TON_TAI"); got != DefaultMessage(DefaultCode()) {
		t.Fatalf("mã lạ phải fallback về %q, nhận %q", DefaultCode(), got)
	}
}

func TestErrorCarriesErrorInfoDetail(t *testing.T) {
	err := Error(TaskTitleRequired)
	st := status.Convert(err)
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("grpc code = %v, muốn InvalidArgument", st.Code())
	}
	wantMessage := DefaultMessage(TaskTitleRequired)
	if st.Message() != wantMessage {
		t.Fatalf("message = %q, muốn %q", st.Message(), wantMessage)
	}
	if strings.Contains(st.Message(), "bắt buộc") {
		t.Fatalf("message không được hardcode tiếng Việt: %q", st.Message())
	}

	var info *errdetails.ErrorInfo
	for _, detail := range st.Details() {
		if candidate, ok := detail.(*errdetails.ErrorInfo); ok {
			info = candidate
			break
		}
	}
	if info == nil {
		t.Fatalf("status thiếu google.rpc.ErrorInfo detail")
	}
	if info.GetReason() != TaskTitleRequired {
		t.Fatalf("reason = %q, muốn %q", info.GetReason(), TaskTitleRequired)
	}
	if info.GetDomain() != Domain {
		t.Fatalf("domain = %q, muốn %q", info.GetDomain(), Domain)
	}

	raw, err := protojson.Marshal(st.Proto())
	if err != nil {
		t.Fatalf("protojson marshal: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, TaskTitleRequired) {
		t.Fatalf("JSON không chứa mã lỗi: %s", body)
	}
	if !strings.Contains(body, "google.rpc.ErrorInfo") {
		t.Fatalf("JSON không chứa @type ErrorInfo: %s", body)
	}
}
