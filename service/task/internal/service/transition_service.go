package service

import (
	"context"

	"taskmanager/service/task/internal/model"
)

type statusTransitionService struct {
	transitions TransitionRepository
	statuses    StatusRepository
}

func NewStatusTransitionService(transitions TransitionRepository, statuses StatusRepository) StatusTransitionService {
	return &statusTransitionService{transitions: transitions, statuses: statuses}
}

func (s *statusTransitionService) Create(ctx context.Context, t model.Transition) (model.Transition, error) {
	from, err := s.statuses.Get(ctx, t.FromStatusID)
	if err != nil {
		return model.Transition{}, err
	}
	to, err := s.statuses.Get(ctx, t.ToStatusID)
	if err != nil {
		return model.Transition{}, err
	}
	if from.ProfileID != t.ProfileID || to.ProfileID != t.ProfileID {
		return model.Transition{}, model.NewError(model.ErrorKindStatusNotInProfile, "status không thuộc profile này")
	}

	t.IsActive = true
	return s.transitions.Create(ctx, t)
}

func (s *statusTransitionService) Get(ctx context.Context, id string) (model.Transition, error) {
	return s.transitions.Get(ctx, id)
}

func (s *statusTransitionService) List(ctx context.Context, profileID, fromStatusID string) ([]model.Transition, error) {
	return s.transitions.List(ctx, profileID, fromStatusID)
}

func (s *statusTransitionService) Update(ctx context.Context, id string, upd model.TransitionUpdate) (model.Transition, error) {
	return s.transitions.Update(ctx, id, upd)
}

func (s *statusTransitionService) Delete(ctx context.Context, id string) error {
	return s.transitions.Delete(ctx, id)
}

func (s *statusTransitionService) ValidateStatusTransition(ctx context.Context, profileID, fromStatusID, toStatusID string) (bool, string, error) {
	allowed, err := s.transitions.IsAllowed(ctx, profileID, fromStatusID, toStatusID)
	if err != nil {
		return false, "", err
	}
	if allowed {
		return true, "rule tồn tại và đang active", nil
	}
	return false, "không có rule chuyển trạng thái (hoặc rule đang tắt)", nil
}
