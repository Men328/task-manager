package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"taskmanager/service/message/internal/model"
	"taskmanager/service/message/internal/service"
)

const (
	alice = "a0000000-0000-0000-0000-000000000001"
	bob   = "b0000000-0000-0000-0000-000000000002"
	carol = "c0000000-0000-0000-0000-000000000003"
)

type fakeFriends struct {
	pairs map[string]bool
	err   error
	calls int
}

func (f *fakeFriends) AreFriends(_ context.Context, a, b string) (bool, error) {
	f.calls++
	if f.err != nil {
		return false, f.err
	}
	first, second := service.OrderedPair(a, b)
	return f.pairs[first+":"+second], nil
}

type fakeBroadcaster struct {
	sent []model.Message
	err  error
}

func (b *fakeBroadcaster) Broadcast(_ context.Context, message model.Message) error {
	b.sent = append(b.sent, message)
	return b.err
}

type fakePublisher struct {
	published []model.Message
	order     *[]string
	err       error
}

func (p *fakePublisher) Publish(_ context.Context, message model.Message) error {
	if p.order != nil {
		*p.order = append(*p.order, "kafka")
	}
	p.published = append(p.published, message)
	return p.err
}

type orderedBroadcaster struct {
	fakeBroadcaster
	order *[]string
}

func (b *orderedBroadcaster) Broadcast(ctx context.Context, message model.Message) error {
	*b.order = append(*b.order, "soketi")
	return b.fakeBroadcaster.Broadcast(ctx, message)
}

type fakeReader struct {
	messages      []model.Message
	conversations []model.Conversation
	lastConvID    string
	lastBefore    time.Time
	lastLimit     int
	err           error
}

func (r *fakeReader) ListMessages(_ context.Context, conversationID string, before time.Time, limit int) ([]model.Message, error) {
	r.lastConvID, r.lastBefore, r.lastLimit = conversationID, before, limit
	if r.err != nil {
		return nil, r.err
	}
	if len(r.messages) > limit {
		return r.messages[:limit], nil
	}
	return r.messages, nil
}

func (r *fakeReader) GetMessage(_ context.Context, id string) (model.Message, error) {
	for _, m := range r.messages {
		if m.ID == id {
			return m, nil
		}
	}
	return model.Message{}, model.ErrNotFound
}

func (r *fakeReader) ListConversations(_ context.Context, _ string, _ int) ([]model.Conversation, error) {
	return r.conversations, r.err
}

type fakeTasks struct {
	requests []model.TaskRequest
	err      error
}

func (f *fakeTasks) CreateForMessage(_ context.Context, request model.TaskRequest) (model.TaskRef, error) {
	f.requests = append(f.requests, request)
	if f.err != nil {
		return model.TaskRef{}, f.err
	}
	return model.TaskRef{TaskID: "t1", Title: request.Draft.Title, StatusName: "To do", StatusCategory: "todo"}, nil
}

type fixture struct {
	svc         service.MessageService
	friends     *fakeFriends
	broadcaster *fakeBroadcaster
	publisher   *fakePublisher
	reader      *fakeReader
	tasks       *fakeTasks
}

func newFixture() *fixture {
	first, second := service.OrderedPair(alice, bob)
	f := &fixture{
		friends:     &fakeFriends{pairs: map[string]bool{first + ":" + second: true}},
		broadcaster: &fakeBroadcaster{},
		publisher:   &fakePublisher{},
		reader:      &fakeReader{},
		tasks:       &fakeTasks{},
	}
	f.svc = service.NewMessageService(f.reader, f.friends, f.broadcaster, f.publisher, f.tasks)
	return f
}

func requireKind(t *testing.T, err error, kind model.ErrorKind) {
	t.Helper()
	var domainErr *model.Error
	if !errors.As(err, &domainErr) || domainErr.Kind != kind {
		t.Fatalf("phải ra kind %q, nhận %v", kind, err)
	}
}

func TestSendBroadcastsThenPublishes(t *testing.T) {
	order := []string{}
	broadcaster := &orderedBroadcaster{order: &order}
	publisher := &fakePublisher{order: &order}
	first, second := service.OrderedPair(alice, bob)
	friends := &fakeFriends{pairs: map[string]bool{first + ":" + second: true}}
	svc := service.NewMessageService(&fakeReader{}, friends, broadcaster, publisher, &fakeTasks{})

	message, err := svc.Send(context.Background(), alice, bob, "  xin chào  ", "c-1")
	if err != nil {
		t.Fatalf("gửi: %v", err)
	}
	if strings.Join(order, ",") != "soketi,kafka" {
		t.Fatalf("phải broadcast trước rồi mới đẩy kafka, nhận %v", order)
	}
	if message.Body != "xin chào" || message.SenderID != alice || message.RecipientID != bob || message.ClientID != "c-1" {
		t.Fatalf("tin nhắn sai: %+v", message)
	}
	if message.ID == "" || message.CreatedAt.IsZero() || message.ConversationID != service.ConversationID(alice, bob) {
		t.Fatalf("thiếu id/thời gian/conversation: %+v", message)
	}
	if broadcaster.sent[0] != message || publisher.published[0] != message {
		t.Fatalf("soketi và kafka phải nhận đúng 1 tin nhắn như response")
	}
}

func TestConversationIDIsSymmetric(t *testing.T) {
	if service.ConversationID(alice, bob) != service.ConversationID(bob, alice) {
		t.Fatal("2 chiều phải cùng conversation")
	}
	if service.ConversationID(alice, bob) == service.ConversationID(alice, carol) {
		t.Fatal("cặp khác phải khác conversation")
	}
	if service.ConversationID(strings.ToUpper(alice), bob) != service.ConversationID(alice, bob) {
		t.Fatal("không phân biệt hoa thường")
	}
}

func TestSendRejectsInvalidInputWithoutSideEffects(t *testing.T) {
	cases := []struct {
		name     string
		peer     string
		body     string
		clientID string
		kind     model.ErrorKind
	}{
		{"nội dung trống", bob, "   ", "", model.ErrorKindBodyRequired},
		{"nội dung quá dài", bob, strings.Repeat("á", model.MaxBodyRunes+1), "", model.ErrorKindBodyTooLong},
		{"nhắn cho chính mình", alice, "hi", "", model.ErrorKindCannotMessageSelf},
		{"peer không phải uuid", "bob", "hi", "", model.ErrorKindInvalidArgument},
		{"client id quá dài", bob, "hi", strings.Repeat("x", model.MaxClientIDBytes+1), model.ErrorKindInvalidArgument},
		{"không phải bạn bè", carol, "hi", "", model.ErrorKindNotFriends},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture()
			_, err := f.svc.Send(context.Background(), alice, tc.peer, tc.body, tc.clientID)
			requireKind(t, err, tc.kind)
			if len(f.broadcaster.sent) != 0 || len(f.publisher.published) != 0 {
				t.Fatal("tin nhắn bị từ chối không được broadcast/đẩy kafka")
			}
		})
	}
}

func TestSendAcceptsMaxLengthBody(t *testing.T) {
	f := newFixture()
	if _, err := f.svc.Send(context.Background(), alice, bob, strings.Repeat("á", model.MaxBodyRunes), ""); err != nil {
		t.Fatalf("đúng giới hạn phải được gửi: %v", err)
	}
}

func TestSendPropagatesFriendCheckFailure(t *testing.T) {
	f := newFixture()
	f.friends.err = model.ErrUnavailable

	_, err := f.svc.Send(context.Background(), alice, bob, "hi", "")
	if !errors.Is(err, model.ErrUnavailable) || len(f.broadcaster.sent) != 0 {
		t.Fatalf("lỗi social phải trả về và không broadcast, nhận %v", err)
	}
}

func TestSendStillPublishesWhenBroadcastFails(t *testing.T) {
	f := newFixture()
	f.broadcaster.err = errors.New("soketi down")

	if _, err := f.svc.Send(context.Background(), alice, bob, "hi", ""); err != nil {
		t.Fatalf("soketi lỗi không được làm hỏng thao tác: %v", err)
	}
	if len(f.publisher.published) != 1 {
		t.Fatal("vẫn phải đẩy kafka để lưu tin nhắn")
	}
}

func TestSendFailsWhenKafkaRejects(t *testing.T) {
	f := newFixture()
	f.publisher.err = errors.New("writer closed")

	_, err := f.svc.Send(context.Background(), alice, bob, "hi", "")
	if !errors.Is(err, model.ErrUnavailable) {
		t.Fatalf("không đẩy được kafka phải báo unavailable, nhận %v", err)
	}
}

func TestListMessagesReturnsOldestFirstWithHasMore(t *testing.T) {
	f := newFixture()
	base := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		f.reader.messages = append(f.reader.messages, model.Message{ID: string(rune('c' - i)), CreatedAt: base.Add(-time.Duration(i) * time.Minute)})
	}
	before := base.Add(time.Hour)

	page, hasMore, err := f.svc.ListMessages(context.Background(), alice, bob, before, 2)
	if err != nil {
		t.Fatalf("đọc: %v", err)
	}
	if !hasMore || len(page) != 2 || page[0].ID != "b" || page[1].ID != "c" {
		t.Fatalf("phải trả 2 tin mới nhất theo thứ tự cũ->mới, nhận %+v hasMore=%v", page, hasMore)
	}
	if f.reader.lastConvID != service.ConversationID(alice, bob) || f.reader.lastLimit != 3 || !f.reader.lastBefore.Equal(before) {
		t.Fatalf("truy vấn sai: %s %d %v", f.reader.lastConvID, f.reader.lastLimit, f.reader.lastBefore)
	}
}

func TestListMessagesClampsLimit(t *testing.T) {
	f := newFixture()
	for limit, want := range map[int]int{0: 51, -5: 51, 1000: 101} {
		if _, _, err := f.svc.ListMessages(context.Background(), alice, bob, time.Time{}, limit); err != nil {
			t.Fatalf("đọc: %v", err)
		}
		if f.reader.lastLimit != want {
			t.Fatalf("limit %d: phải truy vấn %d, nhận %d", limit, want, f.reader.lastLimit)
		}
	}
}

func TestListMessagesDoesNotRequireFriendship(t *testing.T) {
	f := newFixture()
	if _, _, err := f.svc.ListMessages(context.Background(), alice, carol, time.Time{}, 10); err != nil {
		t.Fatalf("lịch sử vẫn đọc được sau khi hủy bạn: %v", err)
	}
	if f.friends.calls != 0 {
		t.Fatal("đọc lịch sử không gọi social")
	}
}
