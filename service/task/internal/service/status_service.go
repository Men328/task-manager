package service

import (
	"context"
	"sync"

	"taskmanager/service/task/internal/model"
)

var seedLifecycleMu sync.Mutex

type taskStatusService struct {
	statuses    StatusRepository
	tasks       TaskRepository
	transitions TransitionRepository
}

func NewTaskStatusService(statuses StatusRepository, tasks TaskRepository, transitions TransitionRepository) TaskStatusService {
	return &taskStatusService{statuses: statuses, tasks: tasks, transitions: transitions}
}

func (s *taskStatusService) Create(ctx context.Context, st model.Status) (model.Status, error) {
	return s.statuses.Create(ctx, st)
}

func (s *taskStatusService) Get(ctx context.Context, id string) (model.Status, error) {
	return s.statuses.Get(ctx, id)
}

func (s *taskStatusService) List(ctx context.Context, profileID string, includeArchived bool) ([]model.Status, error) {
	return s.statuses.List(ctx, profileID, includeArchived)
}

func (s *taskStatusService) Update(ctx context.Context, id string, upd model.StatusUpdate) (model.Status, error) {
	return s.statuses.Update(ctx, id, upd)
}

func (s *taskStatusService) Delete(ctx context.Context, id string) error {
	inUse, err := s.tasks.CountByStatus(ctx, id)
	if err != nil {
		return err
	}
	if inUse > 0 {
		return model.NewError(model.ErrorKindStatusInUse,
			"status đang được %d task sử dụng, hãy archive thay vì xoá", inUse)
	}
	return s.statuses.Delete(ctx, id)
}

func (s *taskStatusService) SeedDefaultStatuses(ctx context.Context, profileID string) ([]model.Status, []model.Transition, error) {
	seedLifecycleMu.Lock()
	defer seedLifecycleMu.Unlock()

	existing, err := s.statuses.List(ctx, profileID, true)
	if err != nil {
		return nil, nil, err
	}
	bySlug := make(map[string]model.Status, len(existing))
	for _, st := range existing {
		bySlug[st.Slug] = st
	}

	for _, def := range seedDefaultStatuses {
		if _, ok := bySlug[def.Slug]; ok {
			continue
		}
		created, err := s.statuses.Create(ctx, model.Status{
			ProfileID:  profileID,
			Name:       def.Name,
			Slug:       def.Slug,
			Color:      def.Color,
			Category:   def.Category,
			IsDefault:  def.IsDefault,
			IsTerminal: def.IsTerminal,
			Position:   def.Position,
		})
		if err != nil {
			raced, found := s.statusBySlug(ctx, profileID, def.Slug)
			if !found {
				return nil, nil, err
			}
			bySlug[def.Slug] = raced
			continue
		}
		bySlug[created.Slug] = created
	}

	existingTransitions, err := s.transitions.List(ctx, profileID, "")
	if err != nil {
		return nil, nil, err
	}
	rules := make(map[string]bool, len(existingTransitions))
	for _, t := range existingTransitions {
		rules[t.FromStatusID+"->"+t.ToStatusID] = true
	}

	for _, path := range seedDefaultTransitionPaths {
		from, okFrom := bySlug[path[0]]
		to, okTo := bySlug[path[1]]
		if !okFrom || !okTo {
			continue
		}
		key := from.ID + "->" + to.ID
		if rules[key] {
			continue
		}
		if _, err := s.transitions.Create(ctx, model.Transition{
			ProfileID:    profileID,
			FromStatusID: from.ID,
			ToStatusID:   to.ID,
			IsActive:     true,
		}); err != nil && !s.transitionRuleExists(ctx, profileID, from.ID, to.ID) {
			return nil, nil, err
		}
		rules[key] = true
	}

	statuses, err := s.statuses.List(ctx, profileID, true)
	if err != nil {
		return nil, nil, err
	}
	transitions, err := s.transitions.List(ctx, profileID, "")
	if err != nil {
		return nil, nil, err
	}
	return statuses, transitions, nil
}

func (s *taskStatusService) statusBySlug(ctx context.Context, profileID, slug string) (model.Status, bool) {
	items, err := s.statuses.List(ctx, profileID, true)
	if err != nil {
		return model.Status{}, false
	}
	for _, st := range items {
		if st.Slug == slug {
			return st, true
		}
	}
	return model.Status{}, false
}

func (s *taskStatusService) transitionRuleExists(ctx context.Context, profileID, fromStatusID, toStatusID string) bool {
	items, err := s.transitions.List(ctx, profileID, fromStatusID)
	if err != nil {
		return false
	}
	for _, t := range items {
		if t.ToStatusID == toStatusID {
			return true
		}
	}
	return false
}
