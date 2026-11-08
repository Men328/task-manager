package service_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/worker/internal/model"
	"taskmanager/service/worker/internal/service"
)

type fakeStore struct {
	saved []model.Message
	err   error
}

func (s *fakeStore) Save(_ context.Context, message model.Message) error {
	s.saved = append(s.saved, message)
	return s.err
}

func (s *fakeStore) UpdateTaskStatus(context.Context, model.TaskStatusChanged) (model.Message, bool, error) {
	return model.Message{}, false, nil
}

func validMessage() model.Message {
	return model.Message{ID: "m1", ConversationID: "c1", SenderID: "a", RecipientID: "b", Body: "hi", CreatedAt: time.Now()}
}

func TestPersistSavesValidMessage(t *testing.T) {
	store := &fakeStore{}
	if err := service.NewMessagePersister(store).Persist(context.Background(), validMessage()); err != nil {
		t.Fatalf("persist: %v", err)
	}
	if len(store.saved) != 1 || store.saved[0].ID != "m1" {
		t.Fatalf("phải lưu đúng 1 tin, nhận %+v", store.saved)
	}
}

func TestPersistRejectsMalformedAsPermanent(t *testing.T) {
	mutations := map[string]func(*model.Message){
		"thiếu id":           func(m *model.Message) { m.ID = "" },
		"thiếu conversation": func(m *model.Message) { m.ConversationID = " " },
		"thiếu body":         func(m *model.Message) { m.Body = "" },
		"tự gửi mình":        func(m *model.Message) { m.RecipientID = m.SenderID },
		"thiếu thời gian":    func(m *model.Message) { m.CreatedAt = time.Time{} },
	}
	for name, mutate := range mutations {
		message := validMessage()
		mutate(&message)
		store := &fakeStore{}

		err := service.NewMessagePersister(store).Persist(context.Background(), message)
		if !errors.Is(err, model.ErrPermanent) || len(store.saved) != 0 {
			t.Fatalf("%s: phải là lỗi vĩnh viễn và không lưu, nhận %v", name, err)
		}
	}
}

func TestPersistPropagatesStoreError(t *testing.T) {
	store := &fakeStore{err: errors.New("db down")}
	err := service.NewMessagePersister(store).Persist(context.Background(), validMessage())
	if err == nil || errors.Is(err, model.ErrPermanent) {
		t.Fatalf("lỗi DB là tạm thời, nhận %v", err)
	}
}
