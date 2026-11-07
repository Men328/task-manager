package dependency

import (
	"testing"

	notificationv1 "taskmanager/common/gen/go/notification/v1"
	"taskmanager/service/notification/internal/model"
)

func TestFriendTargetTypeRoundTrip(t *testing.T) {
	const friend = notificationv1.NoticeTargetType_NOTICE_TARGET_TYPE_FRIEND

	stored := targetTypeFromProto(friend)
	if !model.TargetTypeValid(stored) {
		t.Fatalf("loại đích friend phải hợp lệ, nhận %q", stored)
	}
	if got := targetTypeToProto(stored); got != friend {
		t.Fatalf("phải đổi ngược về FRIEND, nhận %v", got)
	}
}
