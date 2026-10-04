package repository

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/notification/internal/model"
)

type InMemoryNoticeRepository struct {
	mu      sync.RWMutex
	notices map[string]model.Notice
}

func NewInMemoryNoticeRepository() *InMemoryNoticeRepository {
	return &InMemoryNoticeRepository{notices: make(map[string]model.Notice)}
}

func (r *InMemoryNoticeRepository) Create(_ context.Context, notice model.Notice) (model.Notice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	if notice.ID == "" {
		notice.ID = uuid.NewString()
	}
	if notice.TargetType == "" {
		notice.TargetType = model.TargetTypeTask
	}
	notice.IsRead = false
	notice.ReadAt = nil
	notice.CreatedAt = now
	r.notices[notice.ID] = notice
	return notice, nil
}

func (r *InMemoryNoticeRepository) Get(_ context.Context, id string) (model.Notice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	notice, ok := r.notices[id]
	if !ok {
		return model.Notice{}, fmt.Errorf("%w: notice %s", model.ErrNotFound, id)
	}
	return notice, nil
}

func (r *InMemoryNoticeRepository) List(_ context.Context, filter model.Filter) ([]model.Notice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Notice, 0, len(r.notices))
	for _, notice := range r.notices {
		if filter.ProfileID != "" && notice.ProfileID != filter.ProfileID {
			continue
		}
		if filter.UnreadOnly && notice.IsRead {
			continue
		}
		out = append(out, notice)
	}
	sortNotices(out)
	if filter.Limit > 0 && len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}

func (r *InMemoryNoticeRepository) CountUnread(_ context.Context, profileID string) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, notice := range r.notices {
		if notice.ProfileID == profileID && !notice.IsRead {
			count++
		}
	}
	return count, nil
}

func (r *InMemoryNoticeRepository) MarkRead(_ context.Context, profileID string, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	markAll := len(ids) == 1 && ids[0] == model.MarkAllID
	targets := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		targets[id] = struct{}{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	marked := 0
	for id, notice := range r.notices {
		if notice.ProfileID != profileID || notice.IsRead {
			continue
		}
		if !markAll {
			if _, ok := targets[id]; !ok {
				continue
			}
		}
		notice.IsRead = true
		notice.ReadAt = &now
		r.notices[id] = notice
		marked++
	}
	return marked, nil
}

func sortNotices(notices []model.Notice) {
	sort.Slice(notices, func(i, j int) bool {
		if !notices[i].CreatedAt.Equal(notices[j].CreatedAt) {
			return notices[i].CreatedAt.After(notices[j].CreatedAt)
		}
		return notices[i].ID > notices[j].ID
	})
}
