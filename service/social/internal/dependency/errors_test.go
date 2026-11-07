package dependency

import (
	"errors"
	"testing"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/status"

	"taskmanager/common/errorcode"
	"taskmanager/service/social/internal/model"
)

func reason(t *testing.T, err error) string {
	t.Helper()
	for _, detail := range status.Convert(err).Details() {
		if info, ok := detail.(*errdetails.ErrorInfo); ok {
			return info.GetReason()
		}
	}
	t.Fatalf("thiếu ErrorInfo: %v", err)
	return ""
}

func TestToGRPCErrorCoversEveryKind(t *testing.T) {
	cases := map[model.ErrorKind]string{
		model.ErrorKindCannotFriendSelf: errorcode.SocialCannotFriendSelf,
		model.ErrorKindProfileNotFound:  errorcode.SocialProfileNotFound,
		model.ErrorKindRequestExists:    errorcode.SocialFriendRequestExists,
		model.ErrorKindAlreadyFriends:   errorcode.SocialAlreadyFriends,
		model.ErrorKindRequestNotFound:  errorcode.SocialFriendRequestNotFound,
		model.ErrorKindFriendNotFound:   errorcode.SocialFriendNotFound,
		model.ErrorKindEmailInvalid:     errorcode.SocialEmailInvalid,
	}
	for kind, want := range cases {
		if got := reason(t, ToGRPCError(model.NewError(kind, "x"))); got != want {
			t.Fatalf("%s: phải ra %s, nhận %s", kind, want, got)
		}
	}
}

func TestToGRPCErrorUnavailable(t *testing.T) {
	if got := reason(t, ToGRPCError(model.ErrUnavailable)); got != errorcode.CommonUnavailable {
		t.Fatalf("phải ra COMMON_UNAVAILABLE, nhận %s", got)
	}
}

func TestToGRPCErrorUnknownIsInternal(t *testing.T) {
	if got := reason(t, ToGRPCError(errors.New("boom"))); got != errorcode.CommonInternal {
		t.Fatalf("phải ra COMMON_INTERNAL, nhận %s", got)
	}
}
