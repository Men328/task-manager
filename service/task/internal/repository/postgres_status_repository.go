package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/task/internal/model"
)

const statusColumns = `id::text, profile_id::text, name, slug, COALESCE(description, ''), COALESCE(color, ''), category::text, is_default, is_terminal, position, is_archived, created_at, updated_at`

type PostgresStatusRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresStatusRepository(pool *pgxpool.Pool) *PostgresStatusRepository {
	return &PostgresStatusRepository{pool: pool}
}

func mapStatusError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			if strings.Contains(pgErr.ConstraintName, "slug") {
				return model.NewError(model.ErrorKindStatusSlugAlreadyExists, "slug %s đã tồn tại", pgErr.ConstraintName)
			}
			return model.ErrAlreadyExists
		case "23503", "22P02":
			return model.ErrNotFound
		}
	}
	return err
}

func scanStatus(row pgx.Row) (model.Status, error) {
	var s model.Status
	var category string
	if err := row.Scan(
		&s.ID,
		&s.ProfileID,
		&s.Name,
		&s.Slug,
		&s.Description,
		&s.Color,
		&category,
		&s.IsDefault,
		&s.IsTerminal,
		&s.Position,
		&s.IsArchived,
		&s.CreatedAt,
		&s.UpdatedAt,
	); err != nil {
		return model.Status{}, mapStatusError(err)
	}
	s.Category = categoryFromDB(category)
	return s, nil
}

func (r *PostgresStatusRepository) Create(ctx context.Context, s model.Status) (model.Status, error) {
	const clearDefault = `
		UPDATE task.task_statuses
		SET is_default = false
		WHERE profile_id = $1::uuid AND is_default`

	const insert = `
		INSERT INTO task.task_statuses
			(id, profile_id, name, slug, description, color, category,
			 is_default, is_terminal, position, is_archived)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid, $3, $4,
			 NULLIF($5, ''), NULLIF($6, ''), $7::task.status_category,
			 $8, $9, $10, $11)
		RETURNING ` + statusColumns

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Status{}, mapStatusError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if s.IsDefault {
		if _, err := tx.Exec(ctx, clearDefault, s.ProfileID); err != nil {
			return model.Status{}, mapStatusError(err)
		}
	}

	created, err := scanStatus(tx.QueryRow(ctx, insert,
		s.ID, s.ProfileID, s.Name, s.Slug, s.Description, s.Color, categoryToDB(s.Category),
		s.IsDefault, s.IsTerminal, s.Position, s.IsArchived))
	if err != nil {
		return model.Status{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Status{}, mapStatusError(err)
	}
	return created, nil
}

func (r *PostgresStatusRepository) Get(ctx context.Context, id string) (model.Status, error) {
	const query = `SELECT ` + statusColumns + ` FROM task.task_statuses WHERE id = $1::uuid`
	return scanStatus(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresStatusRepository) List(ctx context.Context, profileID string, includeArchived bool) ([]model.Status, error) {
	const query = `
		SELECT ` + statusColumns + `
		FROM task.task_statuses
		WHERE ($1::uuid IS NULL OR profile_id = $1::uuid)
		  AND ($2 OR NOT is_archived)
		ORDER BY position, created_at`

	rows, err := r.pool.Query(ctx, query, optionalUUID(profileID), includeArchived)
	if err != nil {
		return nil, mapStatusError(err)
	}
	defer rows.Close()

	out := make([]model.Status, 0)
	for rows.Next() {
		s, err := scanStatus(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, mapStatusError(rows.Err())
}

func (r *PostgresStatusRepository) Update(ctx context.Context, id string, upd model.StatusUpdate) (model.Status, error) {
	const clearDefault = `
		UPDATE task.task_statuses
		SET is_default = false
		WHERE profile_id = (SELECT profile_id FROM task.task_statuses WHERE id = $1::uuid)
		  AND id <> $1::uuid
		  AND is_default`

	const update = `
		UPDATE task.task_statuses
		SET name        = COALESCE($2, name),
		    description = COALESCE($3, description),
		    color       = COALESCE($4, color),
		    category    = COALESCE($5::task.status_category, category),
		    is_default  = COALESCE($6, is_default),
		    is_terminal = COALESCE($7, is_terminal),
		    position    = COALESCE($8, position),
		    is_archived = COALESCE($9, is_archived)
		WHERE id = $1::uuid
		RETURNING ` + statusColumns

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Status{}, mapStatusError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if upd.IsDefault != nil && *upd.IsDefault {
		if _, err := tx.Exec(ctx, clearDefault, id); err != nil {
			return model.Status{}, mapStatusError(err)
		}
	}

	updated, err := scanStatus(tx.QueryRow(ctx, update, id,
		upd.Name, upd.Description, upd.Color, categoryToDBPtr(upd.Category), upd.IsDefault,
		upd.IsTerminal, upd.Position, upd.IsArchived))
	if err != nil {
		return model.Status{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Status{}, mapStatusError(err)
	}
	return updated, nil
}

func (r *PostgresStatusRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM task.task_statuses WHERE id = $1::uuid`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return model.NewError(model.ErrorKindStatusInUse, "status %s đang được task sử dụng", id)
		}
		return mapStatusError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
