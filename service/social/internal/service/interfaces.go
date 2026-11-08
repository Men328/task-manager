package service

import (
	"context"

	"taskmanager/service/social/internal/model"
)

type FriendshipRepository interface {
	Create(ctx context.Context, requesterID, addresseeID string) (model.Friendship, error)
	Get(ctx context.Context, id string) (model.Friendship, error)
	GetByPair(ctx context.Context, a, b string) (model.Friendship, error)
	Accept(ctx context.Context, id, addresseeID string) (model.Friendship, error)
	Delete(ctx context.Context, id string, status model.Status) error
	ListAccepted(ctx context.Context, profileID string) ([]model.Friendship, error)
	ListPending(ctx context.Context, profileID string) ([]model.Friendship, error)
}

type ProfileDirectory interface {
	FindByEmail(ctx context.Context, email string) (model.Profile, error)
	GetByIDs(ctx context.Context, ids []string) ([]model.Profile, error)
}

type FriendNotifier interface {
	FriendRequested(ctx context.Context, to string, from model.Profile, requestID string) error
	FriendAccepted(ctx context.Context, to string, by model.Profile, requestID string) error
}

type FriendService interface {
	Search(ctx context.Context, profileID, email string) (*model.Profile, model.Relation, error)
	ListFriends(ctx context.Context, profileID string) ([]model.Friend, error)
	RemoveFriend(ctx context.Context, profileID, friendID string) error
	ListRequests(ctx context.Context, profileID string) (incoming, outgoing []model.FriendRequest, err error)
	SendRequest(ctx context.Context, profileID, targetID string) (model.Relation, error)
	AcceptRequest(ctx context.Context, profileID, requestID string) error
	DeclineRequest(ctx context.Context, profileID, requestID string) error
	CancelRequest(ctx context.Context, profileID, requestID string) error
	AreFriends(ctx context.Context, profileID, otherID string) (bool, error)
}
