package service

import (
	"context"
	"errors"
	"log/slog"
	"strings"

	"taskmanager/service/social/internal/model"
)

const maxEmailLength = 255

type friendService struct {
	friendships FriendshipRepository
	directory   ProfileDirectory
	notifier    FriendNotifier
}

func NewFriendService(friendships FriendshipRepository, directory ProfileDirectory, notifier FriendNotifier) FriendService {
	return &friendService{friendships: friendships, directory: directory, notifier: notifier}
}

func (s *friendService) Search(ctx context.Context, profileID, email string) (*model.Profile, model.Relation, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if !emailValid(normalized) {
		return nil, model.RelationNone, model.NewError(model.ErrorKindEmailInvalid, "email không hợp lệ")
	}

	found, err := s.directory.FindByEmail(ctx, normalized)
	if errors.Is(err, model.ErrNotFound) {
		return nil, model.RelationNone, nil
	}
	if err != nil {
		return nil, model.RelationNone, err
	}
	if found.ID == profileID {
		return &found, model.RelationSelf, nil
	}

	row, err := s.friendships.GetByPair(ctx, profileID, found.ID)
	if errors.Is(err, model.ErrNotFound) {
		return &found, model.RelationNone, nil
	}
	if err != nil {
		return nil, model.RelationNone, err
	}
	return &found, relationOf(row, profileID), nil
}

func (s *friendService) ListFriends(ctx context.Context, profileID string) ([]model.Friend, error) {
	rows, err := s.friendships.ListAccepted(ctx, profileID)
	if err != nil {
		return nil, err
	}
	profiles, err := s.profilesOf(ctx, rows, profileID)
	if err != nil {
		return nil, err
	}

	friends := make([]model.Friend, 0, len(rows))
	for _, row := range rows {
		other, ok := profiles[otherSide(row, profileID)]
		if !ok {
			continue
		}
		friend := model.Friend{Profile: other}
		if row.AcceptedAt != nil {
			friend.Since = *row.AcceptedAt
		}
		friends = append(friends, friend)
	}
	return friends, nil
}

func (s *friendService) RemoveFriend(ctx context.Context, profileID, friendID string) error {
	row, err := s.friendships.GetByPair(ctx, profileID, friendID)
	if errors.Is(err, model.ErrNotFound) {
		return friendNotFound()
	}
	if err != nil {
		return err
	}
	if row.Status != model.StatusAccepted {
		return friendNotFound()
	}
	if err := s.friendships.Delete(ctx, row.ID, model.StatusAccepted); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return friendNotFound()
		}
		return err
	}
	return nil
}

func (s *friendService) ListRequests(ctx context.Context, profileID string) ([]model.FriendRequest, []model.FriendRequest, error) {
	rows, err := s.friendships.ListPending(ctx, profileID)
	if err != nil {
		return nil, nil, err
	}
	profiles, err := s.profilesOf(ctx, rows, profileID)
	if err != nil {
		return nil, nil, err
	}

	incoming := make([]model.FriendRequest, 0, len(rows))
	outgoing := make([]model.FriendRequest, 0, len(rows))
	for _, row := range rows {
		other, ok := profiles[otherSide(row, profileID)]
		if !ok {
			continue
		}
		request := model.FriendRequest{ID: row.ID, Profile: other, CreatedAt: row.CreatedAt}
		if row.AddresseeID == profileID {
			incoming = append(incoming, request)
		} else {
			outgoing = append(outgoing, request)
		}
	}
	return incoming, outgoing, nil
}

func (s *friendService) SendRequest(ctx context.Context, profileID, targetID string) (model.Relation, error) {
	if profileID == targetID {
		return model.RelationNone, model.NewError(model.ErrorKindCannotFriendSelf, "không thể kết bạn với chính mình")
	}

	sender, err := s.requireProfiles(ctx, profileID, targetID)
	if err != nil {
		return model.RelationNone, err
	}

	existing, err := s.friendships.GetByPair(ctx, profileID, targetID)
	if err == nil {
		return s.resolveExisting(ctx, existing, sender)
	}
	if !errors.Is(err, model.ErrNotFound) {
		return model.RelationNone, err
	}

	created, err := s.friendships.Create(ctx, profileID, targetID)
	if errors.Is(err, model.ErrDuplicate) {
		existing, err := s.friendships.GetByPair(ctx, profileID, targetID)
		if err != nil {
			return model.RelationNone, err
		}
		return s.resolveExisting(ctx, existing, sender)
	}
	if err != nil {
		return model.RelationNone, err
	}

	if err := s.notifier.FriendRequested(ctx, targetID, sender, created.ID); err != nil {
		slog.Warn("gửi thông báo lời mời kết bạn", "request_id", created.ID, "error", err)
	}
	return model.RelationOutgoing, nil
}

func (s *friendService) AcceptRequest(ctx context.Context, profileID, requestID string) error {
	row, err := s.friendships.Accept(ctx, requestID, profileID)
	if errors.Is(err, model.ErrNotFound) {
		return requestNotFound()
	}
	if err != nil {
		return err
	}

	profiles, err := s.directory.GetByIDs(ctx, []string{profileID})
	if err != nil || len(profiles) == 0 {
		slog.Warn("tra hồ sơ để báo chấp nhận kết bạn", "request_id", row.ID, "error", err)
		return nil
	}
	s.notifyAccepted(ctx, row, profiles[0])
	return nil
}

func (s *friendService) DeclineRequest(ctx context.Context, profileID, requestID string) error {
	return s.deletePending(ctx, requestID, func(row model.Friendship) bool {
		return row.AddresseeID == profileID
	})
}

func (s *friendService) CancelRequest(ctx context.Context, profileID, requestID string) error {
	return s.deletePending(ctx, requestID, func(row model.Friendship) bool {
		return row.RequesterID == profileID
	})
}

func (s *friendService) deletePending(ctx context.Context, requestID string, allowed func(model.Friendship) bool) error {
	row, err := s.friendships.Get(ctx, requestID)
	if errors.Is(err, model.ErrNotFound) {
		return requestNotFound()
	}
	if err != nil {
		return err
	}
	if row.Status != model.StatusPending || !allowed(row) {
		return requestNotFound()
	}
	if err := s.friendships.Delete(ctx, row.ID, model.StatusPending); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return requestNotFound()
		}
		return err
	}
	return nil
}

func (s *friendService) requireProfiles(ctx context.Context, profileID, targetID string) (model.Profile, error) {
	profiles, err := s.directory.GetByIDs(ctx, []string{profileID, targetID})
	if err != nil {
		return model.Profile{}, err
	}

	var sender model.Profile
	targetFound := false
	for _, p := range profiles {
		switch p.ID {
		case profileID:
			sender = p
		case targetID:
			targetFound = true
		}
	}
	if sender.ID == "" || !targetFound {
		return model.Profile{}, model.NewError(model.ErrorKindProfileNotFound, "không tìm thấy người dùng")
	}
	return sender, nil
}

func (s *friendService) resolveExisting(ctx context.Context, row model.Friendship, sender model.Profile) (model.Relation, error) {
	if row.Status == model.StatusAccepted {
		return model.RelationNone, model.NewError(model.ErrorKindAlreadyFriends, "hai người đã là bạn bè")
	}
	if row.RequesterID == sender.ID {
		return model.RelationNone, model.NewError(model.ErrorKindRequestExists, "đã gửi lời mời")
	}

	accepted, err := s.friendships.Accept(ctx, row.ID, sender.ID)
	if errors.Is(err, model.ErrNotFound) {
		return model.RelationNone, requestNotFound()
	}
	if err != nil {
		return model.RelationNone, err
	}
	s.notifyAccepted(ctx, accepted, sender)
	return model.RelationFriend, nil
}

func (s *friendService) notifyAccepted(ctx context.Context, row model.Friendship, by model.Profile) {
	if err := s.notifier.FriendAccepted(ctx, row.RequesterID, by, row.ID); err != nil {
		slog.Warn("gửi thông báo chấp nhận kết bạn", "request_id", row.ID, "error", err)
	}
}

func (s *friendService) profilesOf(ctx context.Context, rows []model.Friendship, profileID string) (map[string]model.Profile, error) {
	if len(rows) == 0 {
		return map[string]model.Profile{}, nil
	}
	ids := make([]string, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, otherSide(row, profileID))
	}
	profiles, err := s.directory.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	byID := make(map[string]model.Profile, len(profiles))
	for _, p := range profiles {
		byID[p.ID] = p
	}
	return byID, nil
}

func otherSide(row model.Friendship, profileID string) string {
	if row.RequesterID == profileID {
		return row.AddresseeID
	}
	return row.RequesterID
}

func relationOf(row model.Friendship, profileID string) model.Relation {
	switch {
	case row.Status == model.StatusAccepted:
		return model.RelationFriend
	case row.RequesterID == profileID:
		return model.RelationOutgoing
	default:
		return model.RelationIncoming
	}
}

func emailValid(email string) bool {
	if email == "" || len(email) > maxEmailLength {
		return false
	}
	local, domain, found := strings.Cut(email, "@")
	return found && local != "" && domain != "" && !strings.Contains(domain, "@")
}

func requestNotFound() error {
	return model.NewError(model.ErrorKindRequestNotFound, "không tìm thấy lời mời")
}

func friendNotFound() error {
	return model.NewError(model.ErrorKindFriendNotFound, "không phải bạn bè")
}
