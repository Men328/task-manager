package service

import (
	"context"

	"taskmanager/service/social/internal/model"
)

type FriendshipRepository interface {
	// Create trả model.ErrDuplicate khi cặp này đã có dòng, bất kể chiều nào.
	Create(ctx context.Context, requesterID, addresseeID string) (model.Friendship, error)
	Get(ctx context.Context, id string) (model.Friendship, error)
	GetByPair(ctx context.Context, a, b string) (model.Friendship, error)
	// Accept chỉ đổi dòng còn pending của đúng người nhận, nếu không trả model.ErrNotFound.
	Accept(ctx context.Context, id, addresseeID string) (model.Friendship, error)
	// Delete chỉ xoá khi dòng còn ở status, để rút lời mời không xoá nhầm tình bạn vừa được chấp nhận.
	Delete(ctx context.Context, id string, status model.Status) error
	ListAccepted(ctx context.Context, profileID string) ([]model.Friendship, error)
	// ListPending trả lời mời ở cả 2 chiều, mới nhất trước.
	ListPending(ctx context.Context, profileID string) ([]model.Friendship, error)
}

type ProfileDirectory interface {
	// FindByEmail trả model.ErrNotFound khi không có, model.ErrUnavailable khi lỗi kết nối.
	FindByEmail(ctx context.Context, email string) (model.Profile, error)
	// GetByIDs bỏ qua id không tồn tại hoặc đã xoá mềm.
	GetByIDs(ctx context.Context, ids []string) ([]model.Profile, error)
}

type FriendNotifier interface {
	FriendRequested(ctx context.Context, to string, from model.Profile, requestID string) error
	FriendAccepted(ctx context.Context, to string, by model.Profile, requestID string) error
}

type FriendService interface {
	// Search trả profile nil khi không có tài khoản khớp.
	Search(ctx context.Context, profileID, email string) (*model.Profile, model.Relation, error)
	ListFriends(ctx context.Context, profileID string) ([]model.Friend, error)
	RemoveFriend(ctx context.Context, profileID, friendID string) error
	ListRequests(ctx context.Context, profileID string) (incoming, outgoing []model.FriendRequest, err error)
	SendRequest(ctx context.Context, profileID, targetID string) (model.Relation, error)
	AcceptRequest(ctx context.Context, profileID, requestID string) error
	DeclineRequest(ctx context.Context, profileID, requestID string) error
	CancelRequest(ctx context.Context, profileID, requestID string) error
}
