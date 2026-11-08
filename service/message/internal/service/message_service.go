package service

import (
	"context"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"taskmanager/service/message/internal/model"
)

const (
	defaultPageSize   = 50
	maxPageSize       = 100
	conversationLimit = 100
)

type messageService struct {
	reader      MessageReader
	friends     FriendChecker
	broadcaster Broadcaster
	publisher   EventPublisher
	now         func() time.Time
}

func NewMessageService(reader MessageReader, friends FriendChecker, broadcaster Broadcaster, publisher EventPublisher) MessageService {
	return &messageService{
		reader:      reader,
		friends:     friends,
		broadcaster: broadcaster,
		publisher:   publisher,
		now:         time.Now,
	}
}

func (s *messageService) Send(ctx context.Context, profileID, peerID, body, clientID string) (model.Message, error) {
	peer, err := s.normalizePeer(profileID, peerID)
	if err != nil {
		return model.Message{}, err
	}
	text, err := normalizeBody(body)
	if err != nil {
		return model.Message{}, err
	}
	clientID = strings.TrimSpace(clientID)
	if len(clientID) > model.MaxClientIDBytes {
		return model.Message{}, model.NewError(model.ErrorKindInvalidArgument, "client_id quá dài")
	}

	friends, err := s.friends.AreFriends(ctx, profileID, peer)
	if err != nil {
		return model.Message{}, err
	}
	if !friends {
		return model.Message{}, model.NewError(model.ErrorKindNotFriends, "chỉ nhắn tin được với bạn bè")
	}

	id, err := uuid.NewV7()
	if err != nil {
		return model.Message{}, err
	}
	message := model.Message{
		ID:             id.String(),
		ConversationID: ConversationID(profileID, peer),
		SenderID:       profileID,
		RecipientID:    peer,
		Body:           text,
		ClientID:       clientID,
		CreatedAt:      s.now().UTC().Truncate(time.Microsecond),
	}

	if err := s.broadcaster.Broadcast(ctx, message); err != nil {
		slog.Warn("broadcast tin nhắn qua soketi thất bại", "message_id", message.ID, "error", err)
	}
	if err := s.publisher.Publish(ctx, message); err != nil {
		slog.Error("đẩy tin nhắn vào kafka thất bại", "message_id", message.ID, "error", err)
		return model.Message{}, model.ErrUnavailable
	}
	return message, nil
}

func (s *messageService) ListMessages(ctx context.Context, profileID, peerID string, before time.Time, limit int) ([]model.Message, bool, error) {
	peer, err := s.normalizePeer(profileID, peerID)
	if err != nil {
		return nil, false, err
	}
	if limit <= 0 {
		limit = defaultPageSize
	}
	if limit > maxPageSize {
		limit = maxPageSize
	}

	rows, err := s.reader.ListMessages(ctx, ConversationID(profileID, peer), before, limit+1)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > limit
	if hasMore {
		rows = rows[:limit]
	}

	ordered := make([]model.Message, len(rows))
	for i, row := range rows {
		ordered[len(rows)-1-i] = row
	}
	return ordered, hasMore, nil
}

func (s *messageService) ListConversations(ctx context.Context, profileID string) ([]model.Conversation, error) {
	return s.reader.ListConversations(ctx, profileID, conversationLimit)
}

func (s *messageService) normalizePeer(profileID, peerID string) (string, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(peerID))
	if err != nil {
		return "", model.NewError(model.ErrorKindInvalidArgument, "peer_id không hợp lệ")
	}
	peer := parsed.String()
	if strings.EqualFold(peer, profileID) {
		return "", model.NewError(model.ErrorKindCannotMessageSelf, "không thể nhắn tin cho chính mình")
	}
	return peer, nil
}

func normalizeBody(body string) (string, error) {
	text := strings.TrimSpace(body)
	if text == "" {
		return "", model.NewError(model.ErrorKindBodyRequired, "nội dung tin nhắn trống")
	}
	if !utf8.ValidString(text) {
		return "", model.NewError(model.ErrorKindInvalidArgument, "nội dung tin nhắn không phải UTF-8")
	}
	if utf8.RuneCountInString(text) > model.MaxBodyRunes {
		return "", model.NewError(model.ErrorKindBodyTooLong, "tin nhắn dài quá %d ký tự", model.MaxBodyRunes)
	}
	return text, nil
}
