package repository

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/mail-provider/internal/model"
)

const providerGoogle = "google"

const subscriptionColumns = `
	s.profile_id::text,
	s.email,
	COALESCE(s.access_token, ''),
	COALESCE(s.refresh_token, ''),
	s.access_token_expires_at,
	COALESCE(n.history_id, ''),
	n.watch_expires_at,
	s.created_at,
	s.updated_at`

const subscriptionFrom = `
	FROM mail_provider.sessions s
	LEFT JOIN mail_provider.noti_indexes n ON n.profile_id = s.profile_id`

type PostgresSubscriptionRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresSubscriptionRepository(pool *pgxpool.Pool) *PostgresSubscriptionRepository {
	return &PostgresSubscriptionRepository{pool: pool}
}

func mapSubscriptionError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return model.ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return model.ErrNotFound
	}
	return err
}

func nullableTime(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func scanSubscription(row pgx.Row) (model.Subscription, error) {
	var sub model.Subscription
	var accessTokenExpiresAt *time.Time
	var watchExpiresAt *time.Time

	if err := row.Scan(
		&sub.ProfileID,
		&sub.Email,
		&sub.AccessToken,
		&sub.RefreshToken,
		&accessTokenExpiresAt,
		&sub.HistoryID,
		&watchExpiresAt,
		&sub.CreatedAt,
		&sub.UpdatedAt,
	); err != nil {
		return model.Subscription{}, mapSubscriptionError(err)
	}

	if accessTokenExpiresAt != nil {
		sub.AccessTokenExpiresAt = accessTokenExpiresAt.UTC()
	}
	if watchExpiresAt != nil {
		sub.WatchExpiresAt = watchExpiresAt.UTC()
	}
	return sub, nil
}

func writeWatch(ctx context.Context, tx pgx.Tx, profileID string, historyID string, watchExpiresAt time.Time) error {
	const query = `
		INSERT INTO mail_provider.noti_indexes (profile_id, history_id, watch_expires_at, last_notice_at)
		VALUES ($1::uuid, NULLIF($2, ''), $3, now())
		ON CONFLICT (profile_id) DO UPDATE SET
			history_id       = COALESCE(excluded.history_id, mail_provider.noti_indexes.history_id),
			watch_expires_at = COALESCE(excluded.watch_expires_at, mail_provider.noti_indexes.watch_expires_at),
			last_notice_at   = now()`

	_, err := tx.Exec(ctx, query, profileID, strings.TrimSpace(historyID), nullableTime(watchExpiresAt))
	return mapSubscriptionError(err)
}

func (r *PostgresSubscriptionRepository) Upsert(ctx context.Context, sub model.Subscription) (model.Subscription, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Subscription{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const sessionQuery = `
		INSERT INTO mail_provider.sessions (profile_id, provider, email, refresh_token, access_token, access_token_expires_at, revoked_at)
		VALUES ($1::uuid, $2, $3, NULLIF($4, ''), NULLIF($5, ''), $6, NULL)
		ON CONFLICT (profile_id, provider) DO UPDATE SET
			email                   = excluded.email,
			refresh_token           = COALESCE(excluded.refresh_token, mail_provider.sessions.refresh_token),
			access_token            = COALESCE(excluded.access_token, mail_provider.sessions.access_token),
			access_token_expires_at = COALESCE(excluded.access_token_expires_at, mail_provider.sessions.access_token_expires_at),
			revoked_at              = NULL`

	if _, err := tx.Exec(ctx, sessionQuery,
		sub.ProfileID,
		providerGoogle,
		strings.ToLower(strings.TrimSpace(sub.Email)),
		sub.RefreshToken,
		sub.AccessToken,
		nullableTime(sub.AccessTokenExpiresAt),
	); err != nil {
		return model.Subscription{}, mapSubscriptionError(err)
	}

	if err := writeWatch(ctx, tx, sub.ProfileID, sub.HistoryID, sub.WatchExpiresAt); err != nil {
		return model.Subscription{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Subscription{}, err
	}
	return r.GetByProfileID(ctx, sub.ProfileID)
}

func (r *PostgresSubscriptionRepository) GetByProfileID(ctx context.Context, profileID string) (model.Subscription, error) {
	const query = `SELECT ` + subscriptionColumns + subscriptionFrom + `
		WHERE s.profile_id = $1::uuid AND s.revoked_at IS NULL`

	return scanSubscription(r.pool.QueryRow(ctx, query, profileID))
}

func (r *PostgresSubscriptionRepository) GetByEmail(ctx context.Context, email string) (model.Subscription, error) {
	const query = `SELECT ` + subscriptionColumns + subscriptionFrom + `
		WHERE lower(s.email) = lower($1) AND s.revoked_at IS NULL`

	return scanSubscription(r.pool.QueryRow(ctx, query, email))
}

func (r *PostgresSubscriptionRepository) List(ctx context.Context) ([]model.Subscription, error) {
	const query = `SELECT ` + subscriptionColumns + subscriptionFrom + `
		WHERE s.revoked_at IS NULL
		ORDER BY s.created_at`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, mapSubscriptionError(err)
	}
	defer rows.Close()

	out := make([]model.Subscription, 0)
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sub)
	}
	return out, mapSubscriptionError(rows.Err())
}

func (r *PostgresSubscriptionRepository) UpdateTokens(ctx context.Context, profileID string, token model.Token) (model.Subscription, error) {
	const query = `
		WITH updated AS (
			UPDATE mail_provider.sessions
			SET access_token            = NULLIF($2, ''),
			    refresh_token           = COALESCE(NULLIF($3, ''), refresh_token),
			    access_token_expires_at = $4
			WHERE profile_id = $1::uuid AND revoked_at IS NULL
			RETURNING profile_id, email, access_token, refresh_token, access_token_expires_at, created_at, updated_at
		)
		SELECT
			u.profile_id::text,
			u.email,
			COALESCE(u.access_token, ''),
			COALESCE(u.refresh_token, ''),
			u.access_token_expires_at,
			COALESCE(n.history_id, ''),
			n.watch_expires_at,
			u.created_at,
			u.updated_at
		FROM updated u
		LEFT JOIN mail_provider.noti_indexes n ON n.profile_id = u.profile_id`

	return scanSubscription(r.pool.QueryRow(ctx, query, profileID, token.AccessToken, token.RefreshToken, nullableTime(token.ExpiresAt)))
}

func (r *PostgresSubscriptionRepository) UpdateWatch(ctx context.Context, profileID string, historyID string, watchExpiresAt time.Time) (model.Subscription, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Subscription{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const existsQuery = `SELECT count(*) FROM mail_provider.sessions WHERE profile_id = $1::uuid AND revoked_at IS NULL`

	var exists int
	if err := tx.QueryRow(ctx, existsQuery, profileID).Scan(&exists); err != nil {
		return model.Subscription{}, mapSubscriptionError(err)
	}
	if exists == 0 {
		return model.Subscription{}, model.ErrNotFound
	}

	if err := writeWatch(ctx, tx, profileID, historyID, watchExpiresAt); err != nil {
		return model.Subscription{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return model.Subscription{}, err
	}
	return r.GetByProfileID(ctx, profileID)
}

func (r *PostgresSubscriptionRepository) Delete(ctx context.Context, profileID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const revokeQuery = `UPDATE mail_provider.sessions SET revoked_at = now() WHERE profile_id = $1::uuid AND revoked_at IS NULL`

	tag, err := tx.Exec(ctx, revokeQuery, profileID)
	if err != nil {
		return mapSubscriptionError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}

	const clearQuery = `DELETE FROM mail_provider.noti_indexes WHERE profile_id = $1::uuid`
	if _, err := tx.Exec(ctx, clearQuery, profileID); err != nil {
		return mapSubscriptionError(err)
	}
	return tx.Commit(ctx)
}
