package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/task/internal/model"
)

const transitionColumns = `id::text, profile_id::text, from_status_id::text, to_status_id::text, is_active, requires_note, COALESCE(description, ''), created_at, updated_at`

type PostgresTransitionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTransitionRepository(pool *pgxpool.Pool) *PostgresTransitionRepository {
	return &PostgresTransitionRepository{pool: pool}
}

func mapTransitionError(err error) error {
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
			return model.ErrAlreadyExists
		case "23503", "22P02":
			return model.ErrNotFound
		}
	}
	return err
}

func scanTransition(row pgx.Row) (model.Transition, error) {
	var t model.Transition
	if err := row.Scan(
		&t.ID,
		&t.ProfileID,
		&t.FromStatusID,
		&t.ToStatusID,
		&t.IsActive,
		&t.RequiresNote,
		&t.Description,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		return model.Transition{}, mapTransitionError(err)
	}
	return t, nil
}

func (r *PostgresTransitionRepository) Create(ctx context.Context, t model.Transition) (model.Transition, error) {
	const query = `
		INSERT INTO task.status_transitions
			(id, profile_id, from_status_id, to_status_id, is_active, requires_note, description)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid, $3::uuid, $4::uuid,
			 $5, $6, NULLIF($7, ''))
		RETURNING ` + transitionColumns

	return scanTransition(r.pool.QueryRow(ctx, query,
		t.ID, t.ProfileID, t.FromStatusID, t.ToStatusID, t.IsActive, t.RequiresNote, t.Description))
}

func (r *PostgresTransitionRepository) Get(ctx context.Context, id string) (model.Transition, error) {
	const query = `SELECT ` + transitionColumns + ` FROM task.status_transitions WHERE id = $1::uuid`
	return scanTransition(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresTransitionRepository) List(ctx context.Context, profileID, fromStatusID string) ([]model.Transition, error) {
	const query = `
		SELECT ` + transitionColumns + `
		FROM task.status_transitions
		WHERE ($1::uuid IS NULL OR profile_id = $1::uuid)
		  AND ($2::uuid IS NULL OR from_status_id = $2::uuid)
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, optionalUUID(profileID), optionalUUID(fromStatusID))
	if err != nil {
		return nil, mapTransitionError(err)
	}
	defer rows.Close()

	out := make([]model.Transition, 0)
	for rows.Next() {
		t, err := scanTransition(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, mapTransitionError(rows.Err())
}

func (r *PostgresTransitionRepository) Update(ctx context.Context, id string, upd model.TransitionUpdate) (model.Transition, error) {
	const query = `
		UPDATE task.status_transitions
		SET is_active     = COALESCE($2, is_active),
		    requires_note = COALESCE($3, requires_note),
		    description   = COALESCE($4, description)
		WHERE id = $1::uuid
		RETURNING ` + transitionColumns

	return scanTransition(r.pool.QueryRow(ctx, query, id, upd.IsActive, upd.RequiresNote, upd.Description))
}

func (r *PostgresTransitionRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM task.status_transitions WHERE id = $1::uuid`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return mapTransitionError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *PostgresTransitionRepository) IsAllowed(ctx context.Context, profileID, fromStatusID, toStatusID string) (bool, error) {
	const query = `
		SELECT is_active
		FROM task.status_transitions
		WHERE profile_id = $1::uuid AND from_status_id = $2::uuid AND to_status_id = $3::uuid`

	var active bool
	if err := r.pool.QueryRow(ctx, query, profileID, fromStatusID, toStatusID).Scan(&active); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, mapTransitionError(err)
	}
	return active, nil
}
