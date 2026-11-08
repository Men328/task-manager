package handler

import (
	"context"

	socialv1 "taskmanager/common/gen/go/social/v1"
	"taskmanager/service/social/internal/dependency"
	"taskmanager/service/social/internal/service"
)

type FriendHandler struct {
	socialv1.UnimplementedFriendServiceServer
	friends service.FriendService
}

func NewFriendHandler(friends service.FriendService) *FriendHandler {
	return &FriendHandler{friends: friends}
}

func (h *FriendHandler) SearchByEmail(ctx context.Context, req *socialv1.SearchByEmailRequest) (*socialv1.SearchByEmailResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	profile, relation, err := h.friends.Search(ctx, req.GetProfileId(), req.GetEmail())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	response := &socialv1.SearchByEmailResponse{Relation: string(relation)}
	if profile != nil {
		response.Profile = dependency.ProfileToProto(*profile)
	}
	return response, nil
}

func (h *FriendHandler) ListFriends(ctx context.Context, req *socialv1.ListFriendsRequest) (*socialv1.ListFriendsResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	friends, err := h.friends.ListFriends(ctx, req.GetProfileId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &socialv1.ListFriendsResponse{Friends: dependency.FriendsToProto(friends)}, nil
}

func (h *FriendHandler) RemoveFriend(ctx context.Context, req *socialv1.RemoveFriendRequest) (*socialv1.RemoveFriendResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	if err := h.friends.RemoveFriend(ctx, req.GetProfileId(), req.GetFriendId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &socialv1.RemoveFriendResponse{}, nil
}

func (h *FriendHandler) ListFriendRequests(ctx context.Context, req *socialv1.ListFriendRequestsRequest) (*socialv1.ListFriendRequestsResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	incoming, outgoing, err := h.friends.ListRequests(ctx, req.GetProfileId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &socialv1.ListFriendRequestsResponse{
		Incoming: dependency.RequestsToProto(incoming),
		Outgoing: dependency.RequestsToProto(outgoing),
	}, nil
}

func (h *FriendHandler) SendFriendRequest(ctx context.Context, req *socialv1.SendFriendRequestRequest) (*socialv1.SendFriendRequestResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	relation, err := h.friends.SendRequest(ctx, req.GetProfileId(), req.GetTargetId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &socialv1.SendFriendRequestResponse{Relation: string(relation)}, nil
}

func (h *FriendHandler) AcceptFriendRequest(ctx context.Context, req *socialv1.AcceptFriendRequestRequest) (*socialv1.AcceptFriendRequestResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	if err := h.friends.AcceptRequest(ctx, req.GetProfileId(), req.GetRequestId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &socialv1.AcceptFriendRequestResponse{}, nil
}

func (h *FriendHandler) DeclineFriendRequest(ctx context.Context, req *socialv1.DeclineFriendRequestRequest) (*socialv1.DeclineFriendRequestResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	if err := h.friends.DeclineRequest(ctx, req.GetProfileId(), req.GetRequestId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &socialv1.DeclineFriendRequestResponse{}, nil
}

func (h *FriendHandler) CancelFriendRequest(ctx context.Context, req *socialv1.CancelFriendRequestRequest) (*socialv1.CancelFriendRequestResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	if err := h.friends.CancelRequest(ctx, req.GetProfileId(), req.GetRequestId()); err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &socialv1.CancelFriendRequestResponse{}, nil
}

func (h *FriendHandler) AreFriends(ctx context.Context, req *socialv1.AreFriendsRequest) (*socialv1.AreFriendsResponse, error) {
	if err := dependency.ValidateProfileID(req.GetProfileId()); err != nil {
		return nil, err
	}

	friends, err := h.friends.AreFriends(ctx, req.GetProfileId(), req.GetOtherId())
	if err != nil {
		return nil, dependency.ToGRPCError(err)
	}
	return &socialv1.AreFriendsResponse{Friends: friends}, nil
}
