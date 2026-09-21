package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/identity/internal/model"
)

const profileColumns = `id::text, email, display_name, COALESCE(avatar_url, ''), timezone, locale, is_active, last_login_at, created_at, updated_at`

type PostgresProfileRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresProfileRepository(pool *pgxpool.Pool) *PostgresProfileRepository {
	return &PostgresProfileRepository{pool: pool}
}

func mapIdentityError(err error) error {
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
			return model.ErrEmailExists
		case "22P02":
			return model.ErrNotFound
		}
	}
	return err
}

func scanProfile(row pgx.Row) (model.Profile, error) {
	var p model.Profile
	var lastLoginAt *time.Time
	if err := row.Scan(
		&p.ID,
		&p.Email,
		&p.DisplayName,
		&p.AvatarURL,
		&p.Timezone,
		&p.Locale,
		&p.IsActive,
		&lastLoginAt,
		&p.CreatedAt,
		&p.UpdatedAt,
	); err != nil {
		return model.Profile{}, mapIdentityError(err)
	}
	p.LastLoginAt = lastLoginAt
	return p, nil
}

func (r *PostgresProfileRepository) Create(ctx context.Context, p model.Profile) (model.Profile, error) {
	const query = `
		INSERT INTO identity.profiles (email, display_name, avatar_url, timezone, locale, is_active)
		VALUES (lower($1), $2, NULLIF($3, ''), $4, $5, $6)
		RETURNING ` + profileColumns

	row := r.pool.QueryRow(ctx, query, p.Email, p.DisplayName, p.AvatarURL, p.Timezone, p.Locale, p.IsActive)
	return scanProfile(row)
}

func (r *PostgresProfileRepository) Get(ctx context.Context, id string) (model.Profile, error) {
	const query = `SELECT ` + profileColumns + ` FROM identity.profiles WHERE id = $1::uuid AND deleted_at IS NULL`
	return scanProfile(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresProfileRepository) GetByEmail(ctx context.Context, email string) (model.Profile, error) {
	const query = `SELECT ` + profileColumns + ` FROM identity.profiles WHERE lower(email) = lower($1) AND deleted_at IS NULL`
	return scanProfile(r.pool.QueryRow(ctx, query, email))
}

func (r *PostgresProfileRepository) List(ctx context.Context, limit int) ([]model.Profile, error) {
	const query = `SELECT ` + profileColumns + ` FROM identity.profiles WHERE deleted_at IS NULL ORDER BY created_at LIMIT $1`

	rows, err := r.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, mapIdentityError(err)
	}
	defer rows.Close()

	out := make([]model.Profile, 0, limit)
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, mapIdentityError(rows.Err())
}

func (r *PostgresProfileRepository) Update(ctx context.Context, id string, upd model.ProfileUpdate) (model.Profile, error) {
	const query = `
		UPDATE identity.profiles
		SET display_name = COALESCE($2, display_name),
		    avatar_url   = COALESCE($3, avatar_url),
		    timezone     = COALESCE($4, timezone),
		    locale       = COALESCE($5, locale),
		    is_active    = COALESCE($6, is_active)
		WHERE id = $1::uuid AND deleted_at IS NULL
		RETURNING ` + profileColumns

	row := r.pool.QueryRow(ctx, query, id, upd.DisplayName, upd.AvatarURL, upd.Timezone, upd.Locale, upd.IsActive)
	return scanProfile(row)
}

func (r *PostgresProfileRepository) TouchLastLogin(ctx context.Context, id string) (model.Profile, error) {
	const query = `
		UPDATE identity.profiles
		SET last_login_at = now()
		WHERE id = $1::uuid AND deleted_at IS NULL
		RETURNING ` + profileColumns

	return scanProfile(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresProfileRepository) Delete(ctx context.Context, id string) error {
	const query = `UPDATE identity.profiles SET deleted_at = now() WHERE id = $1::uuid AND deleted_at IS NULL`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return mapIdentityError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}
