package model

import "time"

type Status string

const (
	StatusPending  Status = "pending"
	StatusAccepted Status = "accepted"
)

// Relation mô tả quan hệ giữa người gọi và một profile khác.
type Relation string

const (
	RelationNone     Relation = "none"
	RelationSelf     Relation = "self"
	RelationFriend   Relation = "friend"
	RelationOutgoing Relation = "outgoing"
	RelationIncoming Relation = "incoming"
)

// Profile là phần hồ sơ được phép cho người dùng khác thấy.
type Profile struct {
	ID          string
	Email       string
	DisplayName string
	AvatarURL   string
}

type Friendship struct {
	ID          string
	RequesterID string
	AddresseeID string
	Status      Status
	CreatedAt   time.Time
	AcceptedAt  *time.Time
}

type Friend struct {
	Profile Profile
	Since   time.Time
}

// FriendRequest là lời mời đang chờ; Profile là phía bên kia.
type FriendRequest struct {
	ID        string
	Profile   Profile
	CreatedAt time.Time
}
