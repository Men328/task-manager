package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	socialv1 "taskmanager/common/gen/go/social/v1"
	"taskmanager/service/message/internal/model"
)

type stubFriendClient struct {
	socialv1.FriendServiceClient
	friends  bool
	err      error
	request  *socialv1.AreFriendsRequest
	deadline bool
}

func (s *stubFriendClient) AreFriends(ctx context.Context, in *socialv1.AreFriendsRequest, _ ...grpc.CallOption) (*socialv1.AreFriendsResponse, error) {
	s.request = in
	_, s.deadline = ctx.Deadline()
	return &socialv1.AreFriendsResponse{Friends: s.friends}, s.err
}

func TestSocialClientPassesPairAndResult(t *testing.T) {
	stub := &stubFriendClient{friends: true}
	got, err := NewSocialClient(stub, time.Second).AreFriends(context.Background(), "a", "b")

	if err != nil || !got {
		t.Fatalf("phải trả true, nhận %v %v", got, err)
	}
	if stub.request.GetProfileId() != "a" || stub.request.GetOtherId() != "b" || !stub.deadline {
		t.Fatalf("request sai hoặc thiếu deadline: %+v", stub.request)
	}
}

func TestSocialClientMapsUnavailable(t *testing.T) {
	for _, code := range []codes.Code{codes.Unavailable, codes.DeadlineExceeded} {
		stub := &stubFriendClient{err: status.Error(code, "down")}
		_, err := NewSocialClient(stub, time.Second).AreFriends(context.Background(), "a", "b")
		if !errors.Is(err, model.ErrUnavailable) {
			t.Fatalf("%s: phải ra ErrUnavailable, nhận %v", code, err)
		}
	}

	stub := &stubFriendClient{err: status.Error(codes.Internal, "boom")}
	_, err := NewSocialClient(stub, time.Second).AreFriends(context.Background(), "a", "b")
	if err == nil || errors.Is(err, model.ErrUnavailable) {
		t.Fatalf("lỗi khác không phải unavailable, nhận %v", err)
	}
}
