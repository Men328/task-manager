package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"taskmanager/service/notification/internal/model"
)

type stubNoticeRepository struct {
	created   []model.Notice
	markIDs   []string
	markCount int
	markErr   error
	unread    int
	listLimit int
}

func (r *stubNoticeRepository) Create(_ context.Context, notice model.Notice) (model.Notice, error) {
	if notice.ID == "" {
		notice.ID = "notice-1"
	}
	notice.CreatedAt = time.Now().UTC()
	r.created = append(r.created, notice)
	return notice, nil
}

func (r *stubNoticeRepository) Get(_ context.Context, id string) (model.Notice, error) {
	for _, notice := range r.created {
		if notice.ID == id {
			return notice, nil
		}
	}
	return model.Notice{}, model.ErrNotFound
}

func (r *stubNoticeRepository) List(_ context.Context, filter model.Filter) ([]model.Notice, error) {
	r.listLimit = filter.Limit
	return r.created, nil
}

func (r *stubNoticeRepository) CountUnread(_ context.Context, _ string) (int, error) {
	return r.unread, nil
}

func (r *stubNoticeRepository) MarkRead(_ context.Context, _ string, ids []string) (int, error) {
	r.markIDs = ids
	return r.markCount, r.markErr
}

type stubPublisher struct {
	published []model.Notice
	err       error
}

func (p *stubPublisher) Publish(_ context.Context, notice model.Notice) error {
	p.published = append(p.published, notice)
	return p.err
}

func TestNoticeServiceCreatePublishesNotice(t *testing.T) {
	repo := &stubNoticeRepository{}
	publisher := &stubPublisher{}
	svc := NewNoticeService(repo, publisher)

	created, err := svc.Create(context.Background(), model.Notice{
		ProfileID:  "profile-1",
		Type:       model.TypeTaskCreated,
		Title:      "Nộp báo cáo",
		TargetType: model.TargetTypeTask,
		TargetID:   "task-1",
	})
	if err != nil {
		t.Fatalf("Create lỗi: %v", err)
	}
	if created.ID != "notice-1" {
		t.Fatalf("id = %q, muốn notice-1", created.ID)
	}
	if len(publisher.published) != 1 || publisher.published[0].ID != created.ID {
		t.Fatalf("notice chưa được publish: %+v", publisher.published)
	}
}

func TestNoticeServiceCreateKeepsNoticeWhenPublishFails(t *testing.T) {
	repo := &stubNoticeRepository{}
	publisher := &stubPublisher{err: errors.New("soketi down")}
	svc := NewNoticeService(repo, publisher)

	created, err := svc.Create(context.Background(), model.Notice{
		ProfileID:  "profile-1",
		Type:       model.TypeEventCreated,
		Title:      "Họp nhóm",
		TargetType: model.TargetTypeEvent,
		TargetID:   "event-1",
	})
	if err != nil {
		t.Fatalf("Create phải thành công dù publish lỗi, nhận %v", err)
	}
	if created.ID == "" {
		t.Fatal("notice phải được lưu")
	}
}

func TestNoticeServiceMarkReadNormalizesMarkAll(t *testing.T) {
	repo := &stubNoticeRepository{markCount: 3}
	svc := NewNoticeService(repo, nil)

	if _, err := svc.MarkRead(context.Background(), "profile-1", []string{"a", " * ", "b"}); err != nil {
		t.Fatalf("MarkRead lỗi: %v", err)
	}
	if len(repo.markIDs) != 1 || repo.markIDs[0] != model.MarkAllID {
		t.Fatalf("ids = %+v, muốn [%q]", repo.markIDs, model.MarkAllID)
	}
}

func TestNoticeServiceListClampsLimit(t *testing.T) {
	repo := &stubNoticeRepository{}
	svc := NewNoticeService(repo, nil)

	if _, err := svc.List(context.Background(), model.Filter{ProfileID: "profile-1", Limit: 1000}); err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if repo.listLimit != model.MaxLimit {
		t.Fatalf("limit = %d, muốn %d", repo.listLimit, model.MaxLimit)
	}

	if _, err := svc.List(context.Background(), model.Filter{ProfileID: "profile-1"}); err != nil {
		t.Fatalf("List lỗi: %v", err)
	}
	if repo.listLimit != model.DefaultLimit {
		t.Fatalf("limit = %d, muốn %d", repo.listLimit, model.DefaultLimit)
	}
}
