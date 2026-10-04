package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/notification/internal/model"
)

const noticeColumns = `id::text, profile_id::text, type, title, COALESCE(body, ''), target_type, target_id::text, COALESCE(source, ''), is_read, created_at, read_at`

type PostgresNoticeRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresNoticeRepository(pool *pgxpool.Pool) *PostgresNoticeRepository {
	return &PostgresNoticeRepository{pool: pool}
}

func mapNoticeError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23503", "22P02":
			return model.ErrInvalid
		case "23514":
			return model.ErrInvalid
		}
	}
	return err
}

func scanNotice(row pgx.Row) (model.Notice, error) {
	var notice model.Notice
	if err := row.Scan(
		&notice.ID,
		&notice.ProfileID,
		&notice.Type,
		&notice.Title,
		&notice.Body,
		&notice.TargetType,
		&notice.TargetID,
		&notice.Source,
		&notice.IsRead,
		&notice.CreatedAt,
		&notice.ReadAt,
	); err != nil {
		return model.Notice{}, mapNoticeError(err)
	}
	return notice, nil
}

func (r *PostgresNoticeRepository) Create(ctx context.Context, notice model.Notice) (model.Notice, error) {
	const query = `
		INSERT INTO notification.notices
			(id, profile_id, type, title, body, target_type, target_id, source, is_read)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid, $3, $4,
			 NULLIF($5, ''), $6, $7::uuid, NULLIF($8, ''), false)
		RETURNING ` + noticeColumns

	return scanNotice(r.pool.QueryRow(ctx, query,
		notice.ID,
		notice.ProfileID,
		notice.Type,
		notice.Title,
		notice.Body,
		notice.TargetType,
		notice.TargetID,
		notice.Source,
	))
}

func (r *PostgresNoticeRepository) Get(ctx context.Context, id string) (model.Notice, error) {
	const query = `SELECT ` + noticeColumns + ` FROM notification.notices WHERE id = $1::uuid`
	return scanNotice(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresNoticeRepository) List(ctx context.Context, filter model.Filter) ([]model.Notice, error) {
	const query = `
		SELECT ` + noticeColumns + `
		FROM notification.notices
		WHERE profile_id = $1::uuid
		  AND ($2::boolean = false OR is_read = false)
		ORDER BY created_at DESC, id DESC
		LIMIT $3`

	limit := filter.Limit
	if limit <= 0 {
		limit = model.DefaultLimit
	}

	rows, err := r.pool.Query(ctx, query, filter.ProfileID, filter.UnreadOnly, limit)
	if err != nil {
		return nil, mapNoticeError(err)
	}
	defer rows.Close()

	out := make([]model.Notice, 0)
	for rows.Next() {
		notice, err := scanNotice(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, notice)
	}
	return out, mapNoticeError(rows.Err())
}

func (r *PostgresNoticeRepository) CountUnread(ctx context.Context, profileID string) (int, error) {
	const query = `SELECT count(*) FROM notification.notices WHERE profile_id = $1::uuid AND is_read = false`

	var count int
	if err := r.pool.QueryRow(ctx, query, profileID).Scan(&count); err != nil {
		return 0, mapNoticeError(err)
	}
	return count, nil
}

func (r *PostgresNoticeRepository) MarkRead(ctx context.Context, profileID string, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	if len(ids) == 1 && ids[0] == model.MarkAllID {
		const query = `
			UPDATE notification.notices
			SET is_read = true, read_at = now()
			WHERE profile_id = $1::uuid AND is_read = false`

		tag, err := r.pool.Exec(ctx, query, profileID)
		if err != nil {
			return 0, mapNoticeError(err)
		}
		return int(tag.RowsAffected()), nil
	}

	const query = `
		UPDATE notification.notices
		SET is_read = true, read_at = now()
		WHERE profile_id = $1::uuid
		  AND is_read = false
		  AND id::text = ANY($2::text[])`

	tag, err := r.pool.Exec(ctx, query, profileID, ids)
	if err != nil {
		return 0, mapNoticeError(err)
	}
	return int(tag.RowsAffected()), nil
}
