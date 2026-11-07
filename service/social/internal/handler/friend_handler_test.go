package handler

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	socialv1 "taskmanager/common/gen/go/social/v1"
	"taskmanager/service/social/internal/model"
	"taskmanager/service/social/internal/service"
)

type stubFriendService struct {
	service.FriendService
	profile  *model.Profile
	relation model.Relation
}

func (s *stubFriendService) Search(context.Context, string, string) (*model.Profile, model.Relation, error) {
	return s.profile, s.relation, nil
}

func TestHandlerRejectsMissingProfileID(t *testing.T) {
	h := NewFriendHandler(&stubFriendService{})

	_, err := h.ListFriends(context.Background(), &socialv1.ListFriendsRequest{})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("thiếu profile_id phải ra InvalidArgument, nhận %v", err)
	}
}

func TestHandlerSearchUnsetProfileWhenNotFound(t *testing.T) {
	h := NewFriendHandler(&stubFriendService{relation: model.RelationNone})

	got, err := h.SearchByEmail(context.Background(), &socialv1.SearchByEmailRequest{ProfileId: "a", Email: "x@y.z"})
	if err != nil {
		t.Fatalf("tìm: %v", err)
	}
	if got.Profile != nil || got.GetRelation() != "none" {
		t.Fatalf("phải không có profile, nhận %+v", got)
	}
}

func TestHandlerSearchReturnsProfile(t *testing.T) {
	h := NewFriendHandler(&stubFriendService{
		profile:  &model.Profile{ID: "b", Email: "b@x.io", DisplayName: "B", AvatarURL: "b.png"},
		relation: model.RelationFriend,
	})

	got, err := h.SearchByEmail(context.Background(), &socialv1.SearchByEmailRequest{ProfileId: "a", Email: "b@x.io"})
	if err != nil {
		t.Fatalf("tìm: %v", err)
	}
	if got.GetProfile().GetDisplayName() != "B" || got.GetProfile().GetAvatarUrl() != "b.png" || got.GetRelation() != "friend" {
		t.Fatalf("response sai: %+v", got)
	}
}
