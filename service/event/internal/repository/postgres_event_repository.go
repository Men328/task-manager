package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/event/internal/model"
)

const eventColumns = `id::text, profile_id::text, title, COALESCE(description, ''), COALESCE(location, ''), start_at, end_at, all_day, COALESCE(color, ''), status, COALESCE(source, ''), created_at, updated_at`

type PostgresEventRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresEventRepository(pool *pgxpool.Pool) *PostgresEventRepository {
	return &PostgresEventRepository{pool: pool}
}

func mapEventError(err error) error {
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
			switch {
			case strings.Contains(pgErr.ConstraintName, "time_range"):
				return model.NewError(model.ErrorKindTimeRangeInvalid, "end time must not be before start time")
			case strings.Contains(pgErr.ConstraintName, "status"):
				return model.NewError(model.ErrorKindStatusInvalid, "unsupported event status")
			default:
				return model.ErrInvalid
			}
		}
	}
	return err
}

func scanEvent(row pgx.Row) (model.Event, error) {
	var e model.Event
	var status string
	if err := row.Scan(
		&e.ID,
		&e.ProfileID,
		&e.Title,
		&e.Description,
		&e.Location,
		&e.StartAt,
		&e.EndAt,
		&e.AllDay,
		&e.Color,
		&status,
		&e.Source,
		&e.CreatedAt,
		&e.UpdatedAt,
	); err != nil {
		return model.Event{}, mapEventError(err)
	}
	e.Status = model.Status(status)
	return e, nil
}

func optionalUUID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

func optionalTime(t *time.Time) any {
	if t == nil {
		return nil
	}
	return *t
}

func optionalStatus(status model.Status) any {
	if status == "" {
		return nil
	}
	return string(status)
}

func (r *PostgresEventRepository) Create(ctx context.Context, e model.Event) (model.Event, error) {
	const query = `
		INSERT INTO event.events
			(id, profile_id, title, description, location, start_at, end_at, all_day, color, status, source)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid, $3,
			 NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, NULLIF($9, ''),
			 COALESCE(NULLIF($10, ''), 'PLANNED'), NULLIF($11, ''))
		RETURNING ` + eventColumns

	return scanEvent(r.pool.QueryRow(ctx, query,
		e.ID, e.ProfileID, e.Title, e.Description, e.Location,
		e.StartAt, e.EndAt, e.AllDay, e.Color, string(e.Status), e.Source))
}

func (r *PostgresEventRepository) Get(ctx context.Context, id string) (model.Event, error) {
	const query = `SELECT ` + eventColumns + ` FROM event.events WHERE id = $1::uuid`
	return scanEvent(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresEventRepository) List(ctx context.Context, f model.Filter) ([]model.Event, error) {
	const query = `
		SELECT ` + eventColumns + `
		FROM event.events
		WHERE ($1::uuid IS NULL OR profile_id = $1::uuid)
		  AND ($2::timestamptz IS NULL OR COALESCE(end_at, start_at) >= $2::timestamptz)
		  AND ($3::timestamptz IS NULL OR start_at < $3::timestamptz)
		  AND ($4::text IS NULL OR status = $4::text)
		ORDER BY start_at, created_at`

	rows, err := r.pool.Query(ctx, query,
		optionalUUID(f.ProfileID),
		optionalTime(f.From),
		optionalTime(f.To),
		optionalStatus(f.Status),
	)
	if err != nil {
		return nil, mapEventError(err)
	}
	defer rows.Close()

	out := make([]model.Event, 0)
	for rows.Next() {
		e, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, mapEventError(rows.Err())
}

func (r *PostgresEventRepository) Update(ctx context.Context, id string, upd model.Update) (model.Event, error) {
	const query = `
		UPDATE event.events
		SET title       = COALESCE($2, title),
		    description = COALESCE($3, description),
		    location    = COALESCE($4, location),
		    start_at    = COALESCE($5::timestamptz, start_at),
		    end_at      = COALESCE($6::timestamptz, end_at),
		    all_day     = COALESCE($7, all_day),
		    color       = COALESCE(NULLIF($8, ''), color),
		    status      = COALESCE(NULLIF($9, ''), status)
		WHERE id = $1::uuid
		RETURNING ` + eventColumns

	return scanEvent(r.pool.QueryRow(ctx, query,
		id,
		upd.Title,
		upd.Description,
		upd.Location,
		optionalTime(upd.StartAt),
		optionalTime(upd.EndAt),
		upd.AllDay,
		upd.Color,
		optionalStatusValue(upd.Status),
	))
}

func optionalStatusValue(status *model.Status) any {
	if status == nil {
		return nil
	}
	return string(*status)
}

func (r *PostgresEventRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM event.events WHERE id = $1::uuid`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return mapEventError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
