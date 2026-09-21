package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/identity/internal/model"
)

const authProviderColumns = `id::text, profile_id::text, provider, provider_user_id, created_at`

type PostgresAuthProviderRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresAuthProviderRepository(pool *pgxpool.Pool) *PostgresAuthProviderRepository {
	return &PostgresAuthProviderRepository{pool: pool}
}

func scanAuthProvider(row pgx.Row) (model.AuthProvider, error) {
	var ap model.AuthProvider
	if err := row.Scan(&ap.ID, &ap.ProfileID, &ap.Provider, &ap.ProviderUserID, &ap.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return model.AuthProvider{}, model.ErrAuthProviderNotFound
		}
		return model.AuthProvider{}, mapIdentityError(err)
	}
	return ap, nil
}

func (r *PostgresAuthProviderRepository) Create(ctx context.Context, ap model.AuthProvider) (model.AuthProvider, error) {
	const query = `
		INSERT INTO identity.auth_providers (profile_id, provider, provider_user_id)
		VALUES ($1::uuid, $2, $3)
		ON CONFLICT ON CONSTRAINT uq_auth_providers_provider_uid
		DO UPDATE SET profile_id = EXCLUDED.profile_id
		RETURNING ` + authProviderColumns

	row := r.pool.QueryRow(ctx, query, ap.ProfileID, ap.Provider, ap.ProviderUserID)
	return scanAuthProvider(row)
}

func (r *PostgresAuthProviderRepository) GetByProviderUserID(ctx context.Context, provider string, providerUserID string) (model.AuthProvider, error) {
	const query = `SELECT ` + authProviderColumns + ` FROM identity.auth_providers WHERE provider = $1 AND provider_user_id = $2`
	return scanAuthProvider(r.pool.QueryRow(ctx, query, provider, providerUserID))
}

func (r *PostgresAuthProviderRepository) ListByProfileID(ctx context.Context, profileID string) ([]model.AuthProvider, error) {
	const query = `SELECT ` + authProviderColumns + ` FROM identity.auth_providers WHERE profile_id = $1::uuid ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, profileID)
	if err != nil {
		return nil, mapIdentityError(err)
	}
	defer rows.Close()

	out := make([]model.AuthProvider, 0, 4)
	for rows.Next() {
		ap, err := scanAuthProvider(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ap)
	}
	return out, mapIdentityError(rows.Err())
}
