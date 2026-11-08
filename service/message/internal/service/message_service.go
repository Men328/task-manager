package service

import (
	"context"
	"errors"
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
	tasks       TaskCreator
	now         func() time.Time
}

func NewMessageService(reader MessageReader, friends FriendChecker, broadcaster Broadcaster, publisher EventPublisher, tasks TaskCreator) MessageService {
	return &messageService{
		reader:      reader,
		friends:     friends,
		broadcaster: broadcaster,
		publisher:   publisher,
		tasks:       tasks,
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
	message, err := s.prepare(ctx, profileID, peer, clientID)
	if err != nil {
		return model.Message{}, err
	}
	message.Body = text
	return s.deliver(ctx, message)
}

func (s *messageService) SendTask(ctx context.Context, profileID, peerID string, draft model.TaskDraft, clientID string) (model.Message, error) {
	peer, err := s.normalizePeer(profileID, peerID)
	if err != nil {
		return model.Message{}, err
	}
	draft, err = normalizeDraft(draft)
	if err != nil {
		return model.Message{}, err
	}
	message, err := s.prepare(ctx, profileID, peer, clientID)
	if err != nil {
		return model.Message{}, err
	}

	ref, err := s.tasks.CreateForMessage(ctx, model.TaskRequest{
		RecipientID: peer,
		RequestedBy: profileID,
		MessageID:   message.ID,
		Draft:       draft,
	})
	if err != nil {
		return model.Message{}, err
	}
	message.Body = draft.Title
	message.Task = &ref
	return s.deliver(ctx, message)
}

func (s *messageService) Get(ctx context.Context, profileID, messageID string) (model.Message, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(messageID))
	if err != nil {
		return model.Message{}, model.NewError(model.ErrorKindInvalidArgument, "message_id không hợp lệ")
	}
	message, err := s.reader.GetMessage(ctx, parsed.String())
	if errors.Is(err, model.ErrNotFound) {
		return model.Message{}, model.NewError(model.ErrorKindMessageNotFound, "không tìm thấy tin nhắn")
	}
	if err != nil {
		return model.Message{}, err
	}
	if !strings.EqualFold(message.SenderID, profileID) && !strings.EqualFold(message.RecipientID, profileID) {
		return model.Message{}, model.NewError(model.ErrorKindMessageNotFound, "không tìm thấy tin nhắn")
	}
	return message, nil
}

func (s *messageService) prepare(ctx context.Context, profileID, peer, clientID string) (model.Message, error) {
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
	return model.Message{
		ID:             id.String(),
		ConversationID: ConversationID(profileID, peer),
		SenderID:       profileID,
		RecipientID:    peer,
		ClientID:       clientID,
		CreatedAt:      s.now().UTC().Truncate(time.Microsecond),
	}, nil
}

func (s *messageService) deliver(ctx context.Context, message model.Message) (model.Message, error) {
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

var taskPriorities = map[string]bool{"": true, "low": true, "medium": true, "high": true, "urgent": true}

func normalizeDraft(draft model.TaskDraft) (model.TaskDraft, error) {
	draft.Title = strings.TrimSpace(draft.Title)
	draft.Description = strings.TrimSpace(draft.Description)
	draft.Priority = strings.ToLower(strings.TrimSpace(draft.Priority))
	if draft.Title == "" {
		return model.TaskDraft{}, model.NewError(model.ErrorKindTaskTitleRequired, "thiếu tiêu đề task")
	}
	if !utf8.ValidString(draft.Title) || !utf8.ValidString(draft.Description) {
		return model.TaskDraft{}, model.NewError(model.ErrorKindInvalidArgument, "nội dung task không phải UTF-8")
	}
	if utf8.RuneCountInString(draft.Title) > model.MaxTaskTitleRunes || utf8.RuneCountInString(draft.Description) > model.MaxTaskDescriptionRunes {
		return model.TaskDraft{}, model.NewError(model.ErrorKindInvalidArgument, "tiêu đề hoặc mô tả task quá dài")
	}
	if !taskPriorities[draft.Priority] {
		return model.TaskDraft{}, model.NewError(model.ErrorKindInvalidArgument, "priority không hợp lệ")
	}
	if draft.Priority == "" {
		draft.Priority = "medium"
	}
	return draft, nil
}
