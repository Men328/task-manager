package service

import (
	"context"
	"log/slog"
	"strings"

	"taskmanager/service/notification/internal/model"
)

type noticeService struct {
	notices   NoticeRepository
	publisher NoticePublisher
}

func NewNoticeService(notices NoticeRepository, publisher NoticePublisher) NoticeService {
	return &noticeService{notices: notices, publisher: publisher}
}

func (s *noticeService) Create(ctx context.Context, notice model.Notice) (model.Notice, error) {
	notice.ProfileID = strings.TrimSpace(notice.ProfileID)
	notice.Type = strings.TrimSpace(notice.Type)
	notice.Title = strings.TrimSpace(notice.Title)
	notice.TargetID = strings.TrimSpace(notice.TargetID)

	created, err := s.notices.Create(ctx, notice)
	if err != nil {
		return model.Notice{}, err
	}

	if s.publisher != nil {
		if err := s.publisher.Publish(ctx, created); err != nil {
			slog.Error("đẩy notice qua soketi thất bại",
				"notice_id", created.ID,
				"profile_id", created.ProfileID,
				"error", err,
			)
		}
	}
	return created, nil
}

func (s *noticeService) List(ctx context.Context, filter model.Filter) ([]model.Notice, error) {
	filter.ProfileID = strings.TrimSpace(filter.ProfileID)
	if filter.Limit <= 0 {
		filter.Limit = model.DefaultLimit
	}
	if filter.Limit > model.MaxLimit {
		filter.Limit = model.MaxLimit
	}
	return s.notices.List(ctx, filter)
}

func (s *noticeService) CountUnread(ctx context.Context, profileID string) (int, error) {
	return s.notices.CountUnread(ctx, strings.TrimSpace(profileID))
}

func (s *noticeService) MarkRead(ctx context.Context, profileID string, ids []string) (int, error) {
	return s.notices.MarkRead(ctx, strings.TrimSpace(profileID), normalizeIDs(ids))
}

func normalizeIDs(ids []string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			continue
		}
		if trimmed == model.MarkAllID {
			return []string{model.MarkAllID}
		}
		out = append(out, trimmed)
	}
	return out
}
