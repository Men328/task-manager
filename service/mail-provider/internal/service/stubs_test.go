package service

import (
	"context"
	"sync"
	"time"

	"taskmanager/service/mail-provider/internal/model"
)

type stubSubscriptions struct {
	mu    sync.Mutex
	items map[string]model.Subscription
}

func newStubSubscriptions(items ...model.Subscription) *stubSubscriptions {
	stub := &stubSubscriptions{items: map[string]model.Subscription{}}
	for _, item := range items {
		stub.items[item.ProfileID] = item
	}
	return stub
}

func (s *stubSubscriptions) Upsert(_ context.Context, sub model.Subscription) (model.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	if sub.CreatedAt.IsZero() {
		sub.CreatedAt = now
	}
	sub.UpdatedAt = now
	s.items[sub.ProfileID] = sub
	return sub, nil
}

func (s *stubSubscriptions) GetByProfileID(_ context.Context, profileID string) (model.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, found := s.items[profileID]
	if !found {
		return model.Subscription{}, model.ErrNotFound
	}
	return sub, nil
}

func (s *stubSubscriptions) GetByEmail(_ context.Context, email string) (model.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, sub := range s.items {
		if sub.Email == email {
			return sub, nil
		}
	}
	return model.Subscription{}, model.ErrNotFound
}

func (s *stubSubscriptions) List(_ context.Context) ([]model.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	out := make([]model.Subscription, 0, len(s.items))
	for _, sub := range s.items {
		out = append(out, sub)
	}
	return out, nil
}

func (s *stubSubscriptions) UpdateTokens(_ context.Context, profileID string, token model.Token) (model.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, found := s.items[profileID]
	if !found {
		return model.Subscription{}, model.ErrNotFound
	}
	sub.AccessToken = token.AccessToken
	if token.RefreshToken != "" {
		sub.RefreshToken = token.RefreshToken
	}
	sub.AccessTokenExpiresAt = token.ExpiresAt
	s.items[profileID] = sub
	return sub, nil
}

func (s *stubSubscriptions) UpdateWatch(_ context.Context, profileID string, historyID string, watchExpiresAt time.Time) (model.Subscription, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sub, found := s.items[profileID]
	if !found {
		return model.Subscription{}, model.ErrNotFound
	}
	if historyID != "" {
		sub.HistoryID = historyID
	}
	if !watchExpiresAt.IsZero() {
		sub.WatchExpiresAt = watchExpiresAt
	}
	s.items[profileID] = sub
	return sub, nil
}

func (s *stubSubscriptions) Delete(_ context.Context, profileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, found := s.items[profileID]; !found {
		return model.ErrNotFound
	}
	delete(s.items, profileID)
	return nil
}

type stubGmail struct {
	watchResult      model.WatchResult
	watchErr         error
	stopErr          error
	messageIDs       []string
	latestHistoryID  string
	historyErr       error
	messages         map[string]model.EmailMessage
	messageErr       error
	attachments      map[string][]byte
	attachmentErr    error
	profileHistoryID string
	profileErr       error
}

func (g *stubGmail) Watch(_ context.Context, _ string, _ string, _ []string) (model.WatchResult, error) {
	return g.watchResult, g.watchErr
}

func (g *stubGmail) Stop(_ context.Context, _ string) error {
	return g.stopErr
}

func (g *stubGmail) NewMessageIDs(_ context.Context, _ string, _ string, _ []string) ([]string, string, error) {
	return g.messageIDs, g.latestHistoryID, g.historyErr
}

func (g *stubGmail) Message(_ context.Context, _ string, messageID string) (model.EmailMessage, error) {
	if g.messageErr != nil {
		return model.EmailMessage{}, g.messageErr
	}
	message, found := g.messages[messageID]
	if !found {
		return model.EmailMessage{}, model.ErrGmailFailed
	}
	return message, nil
}

func (g *stubGmail) Attachment(_ context.Context, _ string, _ string, attachmentID string) ([]byte, error) {
	if g.attachmentErr != nil {
		return nil, g.attachmentErr
	}
	data, found := g.attachments[attachmentID]
	if !found {
		return nil, model.ErrGmailFailed
	}
	return data, nil
}

func (g *stubGmail) ProfileHistoryID(_ context.Context, _ string) (string, error) {
	return g.profileHistoryID, g.profileErr
}

type stubRefresher struct {
	token model.Token
	err   error
	calls int
}

func (r *stubRefresher) Refresh(_ context.Context, _ string) (model.Token, error) {
	r.calls++
	return r.token, r.err
}

type stubAnalyzer struct {
	draft model.MailDraft
	err   error
}

func (a *stubAnalyzer) Analyze(_ context.Context, _ model.EmailMessage) (model.MailDraft, error) {
	return a.draft, a.err
}

type stubTasks struct {
	created chan model.TaskInput
	err     error
}

func (t *stubTasks) Create(_ context.Context, in model.TaskInput) (model.TaskRef, error) {
	if t.created != nil {
		t.created <- in
	}
	if t.err != nil {
		return model.TaskRef{}, t.err
	}
	return model.TaskRef{ID: "task-1", Title: in.Title}, nil
}

type stubSchedules struct {
	created chan model.ScheduleInput
	err     error
}

func (s *stubSchedules) Create(_ context.Context, in model.ScheduleInput) (model.ScheduleRef, error) {
	if s.created != nil {
		s.created <- in
	}
	if s.err != nil {
		return model.ScheduleRef{}, s.err
	}
	return model.ScheduleRef{ID: "schedule-1", Title: in.Title}, nil
}

type stubEvents struct {
	created chan model.EventInput
	err     error
}

func (e *stubEvents) Create(_ context.Context, in model.EventInput) (model.EventRef, error) {
	if e.created != nil {
		e.created <- in
	}
	if e.err != nil {
		return model.EventRef{}, e.err
	}
	return model.EventRef{ID: "event-1", Title: in.Title}, nil
}

type stubBacklogs struct {
	created chan model.BacklogInput
	err     error
}

func (b *stubBacklogs) Create(_ context.Context, in model.BacklogInput) (model.BacklogRef, error) {
	if b.created != nil {
		b.created <- in
	}
	if b.err != nil {
		return model.BacklogRef{}, b.err
	}
	return model.BacklogRef{ID: "backlog-1", Title: in.Title}, nil
}

type stubNotices struct {
	created chan model.NoticeInput
	err     error
}

func (n *stubNotices) Notify(_ context.Context, in model.NoticeInput) (model.NoticeRef, error) {
	if n.created != nil {
		n.created <- in
	}
	if n.err != nil {
		return model.NoticeRef{}, n.err
	}
	return model.NoticeRef{ID: "notice-1", Title: in.Title}, nil
}

type stubStorage struct {
	mu      sync.Mutex
	objects map[string][]byte
	err     error
}

func newStubStorage() *stubStorage {
	return &stubStorage{objects: map[string][]byte{}}
}

func (s *stubStorage) Put(_ context.Context, key string, _ string, data []byte) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[key] = data
	return key, nil
}

func (s *stubStorage) stored() map[string][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string][]byte, len(s.objects))
	for key, value := range s.objects {
		out[key] = value
	}
	return out
}
