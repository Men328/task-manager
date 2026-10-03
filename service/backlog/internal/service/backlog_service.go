package service

import (
	"context"

	"taskmanager/service/backlog/internal/model"
)

type backlogService struct {
	items BacklogRepository
}

func NewBacklogService(items BacklogRepository) BacklogService {
	return &backlogService{items: items}
}

func (s *backlogService) Create(ctx context.Context, item model.Item) (model.Item, error) {
	status, err := normalizeStatus(item.Status, model.StatusNew)
	if err != nil {
		return model.Item{}, err
	}
	item.Status = status
	item.Category = normalizeCategory(item.Category)
	return s.items.Create(ctx, item)
}

func (s *backlogService) Get(ctx context.Context, id string) (model.Item, error) {
	return s.items.Get(ctx, id)
}

func (s *backlogService) List(ctx context.Context, f model.Filter) ([]model.Item, error) {
	return s.items.List(ctx, f)
}

func (s *backlogService) Update(ctx context.Context, id string, upd model.Update) (model.Item, error) {
	current, err := s.items.Get(ctx, id)
	if err != nil {
		return model.Item{}, err
	}
	if upd.Status != nil {
		status, err := normalizeStatus(*upd.Status, current.Status)
		if err != nil {
			return model.Item{}, err
		}
		upd.Status = &status
	}
	return s.items.Update(ctx, id, upd)
}

func (s *backlogService) Delete(ctx context.Context, id string) error {
	return s.items.Delete(ctx, id)
}
