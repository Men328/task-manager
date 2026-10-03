package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/calendar/internal/model"
)

const scheduleColumns = `id::text, profile_id::text, title, COALESCE(description, ''), COALESCE(location, ''), start_at, end_at, all_day, COALESCE(color, ''), created_at, updated_at`

type PostgresScheduleRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresScheduleRepository(pool *pgxpool.Pool) *PostgresScheduleRepository {
	return &PostgresScheduleRepository{pool: pool}
}

func mapScheduleError(err error) error {
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
			if strings.Contains(pgErr.ConstraintName, "time_range") {
				return model.NewError(model.ErrorKindTimeRangeInvalid, "end time must not be before start time")
			}
			return model.ErrInvalid
		}
	}
	return err
}

func scanSchedule(row pgx.Row) (model.Schedule, error) {
	var s model.Schedule
	if err := row.Scan(
		&s.ID,
		&s.ProfileID,
		&s.Title,
		&s.Description,
		&s.Location,
		&s.StartAt,
		&s.EndAt,
		&s.AllDay,
		&s.Color,
		&s.CreatedAt,
		&s.UpdatedAt,
	); err != nil {
		return model.Schedule{}, mapScheduleError(err)
	}
	return s, nil
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

func (r *PostgresScheduleRepository) Create(ctx context.Context, s model.Schedule) (model.Schedule, error) {
	const query = `
		INSERT INTO calendar.schedules
			(id, profile_id, title, description, location, start_at, end_at, all_day, color)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid, $3,
			 NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, NULLIF($9, ''))
		RETURNING ` + scheduleColumns

	return scanSchedule(r.pool.QueryRow(ctx, query,
		s.ID, s.ProfileID, s.Title, s.Description, s.Location,
		s.StartAt, s.EndAt, s.AllDay, s.Color))
}

func (r *PostgresScheduleRepository) Get(ctx context.Context, id string) (model.Schedule, error) {
	const query = `SELECT ` + scheduleColumns + ` FROM calendar.schedules WHERE id = $1::uuid`
	return scanSchedule(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresScheduleRepository) List(ctx context.Context, f model.ScheduleFilter) ([]model.Schedule, error) {
	const query = `
		SELECT ` + scheduleColumns + `
		FROM calendar.schedules
		WHERE ($1::uuid IS NULL OR profile_id = $1::uuid)
		  AND ($2::timestamptz IS NULL OR COALESCE(end_at, start_at) >= $2::timestamptz)
		  AND ($3::timestamptz IS NULL OR start_at < $3::timestamptz)
		ORDER BY start_at, created_at`

	rows, err := r.pool.Query(ctx, query,
		optionalUUID(f.ProfileID),
		optionalTime(f.From),
		optionalTime(f.To),
	)
	if err != nil {
		return nil, mapScheduleError(err)
	}
	defer rows.Close()

	out := make([]model.Schedule, 0)
	for rows.Next() {
		s, err := scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, mapScheduleError(rows.Err())
}

func (r *PostgresScheduleRepository) Update(ctx context.Context, id string, upd model.ScheduleUpdate) (model.Schedule, error) {
	const query = `
		UPDATE calendar.schedules
		SET title       = COALESCE($2, title),
		    description = COALESCE($3, description),
		    location    = COALESCE($4, location),
		    start_at    = COALESCE($5::timestamptz, start_at),
		    end_at      = COALESCE($6::timestamptz, end_at),
		    all_day     = COALESCE($7, all_day),
		    color       = COALESCE(NULLIF($8, ''), color)
		WHERE id = $1::uuid
		RETURNING ` + scheduleColumns

	return scanSchedule(r.pool.QueryRow(ctx, query,
		id,
		upd.Title,
		upd.Description,
		upd.Location,
		optionalTime(upd.StartAt),
		optionalTime(upd.EndAt),
		upd.AllDay,
		upd.Color,
	))
}

func (r *PostgresScheduleRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM calendar.schedules WHERE id = $1::uuid`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return mapScheduleError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
