package repository

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"taskmanager/service/social/internal/model"
)

type InMemoryFriendshipRepository struct {
	mu    sync.RWMutex
	items map[string]model.Friendship
}

func NewInMemoryFriendshipRepository() *InMemoryFriendshipRepository {
	return &InMemoryFriendshipRepository{items: make(map[string]model.Friendship)}
}

func samePair(f model.Friendship, a, b string) bool {
	return (f.RequesterID == a && f.AddresseeID == b) || (f.RequesterID == b && f.AddresseeID == a)
}

func (r *InMemoryFriendshipRepository) Create(_ context.Context, requesterID, addresseeID string) (model.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.items {
		if samePair(existing, requesterID, addresseeID) {
			return model.Friendship{}, model.ErrDuplicate
		}
	}
	row := model.Friendship{
		ID:          uuid.NewString(),
		RequesterID: requesterID,
		AddresseeID: addresseeID,
		Status:      model.StatusPending,
		CreatedAt:   time.Now().UTC(),
	}
	r.items[row.ID] = row
	return row, nil
}

func (r *InMemoryFriendshipRepository) Get(_ context.Context, id string) (model.Friendship, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	row, ok := r.items[id]
	if !ok {
		return model.Friendship{}, model.ErrNotFound
	}
	return row, nil
}

func (r *InMemoryFriendshipRepository) GetByPair(_ context.Context, a, b string) (model.Friendship, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, row := range r.items {
		if samePair(row, a, b) {
			return row, nil
		}
	}
	return model.Friendship{}, model.ErrNotFound
}

func (r *InMemoryFriendshipRepository) Accept(_ context.Context, id, addresseeID string) (model.Friendship, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	row, ok := r.items[id]
	if !ok || row.Status != model.StatusPending || row.AddresseeID != addresseeID {
		return model.Friendship{}, model.ErrNotFound
	}
	now := time.Now().UTC()
	row.Status = model.StatusAccepted
	row.AcceptedAt = &now
	r.items[id] = row
	return row, nil
}

func (r *InMemoryFriendshipRepository) Delete(_ context.Context, id string, status model.Status) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	row, ok := r.items[id]
	if !ok || row.Status != status {
		return model.ErrNotFound
	}
	delete(r.items, id)
	return nil
}

func (r *InMemoryFriendshipRepository) ListAccepted(_ context.Context, profileID string) ([]model.Friendship, error) {
	out := r.list(profileID, model.StatusAccepted)
	sort.Slice(out, func(i, j int) bool { return out[i].AcceptedAt.After(*out[j].AcceptedAt) })
	return out, nil
}

func (r *InMemoryFriendshipRepository) ListPending(_ context.Context, profileID string) ([]model.Friendship, error) {
	out := r.list(profileID, model.StatusPending)
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}

func (r *InMemoryFriendshipRepository) list(profileID string, status model.Status) []model.Friendship {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]model.Friendship, 0)
	for _, row := range r.items {
		if row.Status == status && (row.RequesterID == profileID || row.AddresseeID == profileID) {
			out = append(out, row)
		}
	}
	return out
}
