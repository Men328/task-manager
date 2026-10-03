package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/backlog/internal/model"
)

const backlogColumns = `id::text, profile_id::text, title, COALESCE(description, ''), COALESCE(sender, ''), COALESCE(source, ''), category, COALESCE(reason, ''), COALESCE(object_key, ''), status, created_at, updated_at`

type PostgresBacklogRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresBacklogRepository(pool *pgxpool.Pool) *PostgresBacklogRepository {
	return &PostgresBacklogRepository{pool: pool}
}

func mapBacklogError(err error) error {
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
			return model.ErrNotFound
		case "23514":
			if strings.Contains(pgErr.ConstraintName, "status") {
				return model.NewError(model.ErrorKindStatusInvalid, "unsupported backlog status")
			}
			return model.ErrInvalid
		}
	}
	return err
}

func scanBacklog(row pgx.Row) (model.Item, error) {
	var item model.Item
	var status string
	if err := row.Scan(
		&item.ID,
		&item.ProfileID,
		&item.Title,
		&item.Description,
		&item.Sender,
		&item.Source,
		&item.Category,
		&item.Reason,
		&item.ObjectKey,
		&status,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return model.Item{}, mapBacklogError(err)
	}
	item.Status = model.Status(status)
	return item, nil
}

func optionalUUID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

func optionalStatus(status model.Status) any {
	if status == "" {
		return nil
	}
	return string(status)
}

func optionalStatusValue(status *model.Status) any {
	if status == nil {
		return nil
	}
	return string(*status)
}

func optionalText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (r *PostgresBacklogRepository) Create(ctx context.Context, item model.Item) (model.Item, error) {
	const query = `
		INSERT INTO backlog.backlogs
			(id, profile_id, title, description, sender, source, category, reason, object_key, status)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid, $3,
			 NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''),
			 COALESCE(NULLIF($7, ''), 'other'), NULLIF($8, ''), NULLIF($9, ''),
			 COALESCE(NULLIF($10, ''), 'NEW'))
		RETURNING ` + backlogColumns

	return scanBacklog(r.pool.QueryRow(ctx, query,
		item.ID, item.ProfileID, item.Title, item.Description, item.Sender,
		item.Source, item.Category, item.Reason, item.ObjectKey, string(item.Status)))
}

func (r *PostgresBacklogRepository) Get(ctx context.Context, id string) (model.Item, error) {
	const query = `SELECT ` + backlogColumns + ` FROM backlog.backlogs WHERE id = $1::uuid`
	return scanBacklog(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresBacklogRepository) List(ctx context.Context, f model.Filter) ([]model.Item, error) {
	const query = `
		SELECT ` + backlogColumns + `
		FROM backlog.backlogs
		WHERE ($1::uuid IS NULL OR profile_id = $1::uuid)
		  AND ($2::text IS NULL OR status = $2::text)
		  AND ($3::text IS NULL OR category = $3::text)
		ORDER BY created_at DESC, id DESC`

	rows, err := r.pool.Query(ctx, query,
		optionalUUID(f.ProfileID),
		optionalStatus(f.Status),
		optionalText(f.Category),
	)
	if err != nil {
		return nil, mapBacklogError(err)
	}
	defer rows.Close()

	out := make([]model.Item, 0)
	for rows.Next() {
		item, err := scanBacklog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}
	return out, mapBacklogError(rows.Err())
}

func (r *PostgresBacklogRepository) Update(ctx context.Context, id string, upd model.Update) (model.Item, error) {
	const query = `
		UPDATE backlog.backlogs
		SET title       = COALESCE($2, title),
		    description = COALESCE($3, description),
		    reason      = COALESCE($4, reason),
		    status      = COALESCE(NULLIF($5, ''), status)
		WHERE id = $1::uuid
		RETURNING ` + backlogColumns

	return scanBacklog(r.pool.QueryRow(ctx, query,
		id,
		upd.Title,
		upd.Description,
		upd.Reason,
		optionalStatusValue(upd.Status),
	))
}

func (r *PostgresBacklogRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM backlog.backlogs WHERE id = $1::uuid`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return mapBacklogError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
