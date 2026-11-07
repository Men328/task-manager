package dependency

import (
	"strings"

	"google.golang.org/protobuf/types/known/timestamppb"

	"taskmanager/common/errorcode"
	socialv1 "taskmanager/common/gen/go/social/v1"
	"taskmanager/service/social/internal/model"
)

func ValidateProfileID(profileID string) error {
	if strings.TrimSpace(profileID) == "" {
		return errorcode.Error(errorcode.CommonInvalidArgument)
	}
	return nil
}

func ProfileToProto(p model.Profile) *socialv1.FriendProfile {
	return &socialv1.FriendProfile{
		Id:          p.ID,
		Email:       p.Email,
		DisplayName: p.DisplayName,
		AvatarUrl:   p.AvatarURL,
	}
}

func FriendsToProto(friends []model.Friend) []*socialv1.Friend {
	out := make([]*socialv1.Friend, 0, len(friends))
	for _, friend := range friends {
		out = append(out, &socialv1.Friend{
			Profile:      ProfileToProto(friend.Profile),
			FriendsSince: timestamppb.New(friend.Since),
		})
	}
	return out
}

func RequestsToProto(requests []model.FriendRequest) []*socialv1.FriendRequest {
	out := make([]*socialv1.FriendRequest, 0, len(requests))
	for _, request := range requests {
		out = append(out, &socialv1.FriendRequest{
			Id:        request.ID,
			Profile:   ProfileToProto(request.Profile),
			CreatedAt: timestamppb.New(request.CreatedAt),
		})
	}
	return out
}
