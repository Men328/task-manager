package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/social/internal/model"
)

const friendshipColumns = `id::text, requester_id::text, addressee_id::text, status, created_at, accepted_at`

type PostgresFriendshipRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresFriendshipRepository(pool *pgxpool.Pool) *PostgresFriendshipRepository {
	return &PostgresFriendshipRepository{pool: pool}
}

func mapFriendshipError(err error) error {
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
			if pgErr.ConstraintName == "uq_friendships_pair" {
				return model.ErrDuplicate
			}
		case "23503", "22P02":
			return model.ErrNotFound
		}
	}
	return err
}

func scanFriendship(row pgx.Row) (model.Friendship, error) {
	var f model.Friendship
	if err := row.Scan(&f.ID, &f.RequesterID, &f.AddresseeID, &f.Status, &f.CreatedAt, &f.AcceptedAt); err != nil {
		return model.Friendship{}, mapFriendshipError(err)
	}
	return f, nil
}

func (r *PostgresFriendshipRepository) Create(ctx context.Context, requesterID, addresseeID string) (model.Friendship, error) {
	const query = `
		INSERT INTO social.friendships (requester_id, addressee_id)
		VALUES ($1::uuid, $2::uuid)
		RETURNING ` + friendshipColumns
	return scanFriendship(r.pool.QueryRow(ctx, query, requesterID, addresseeID))
}

func (r *PostgresFriendshipRepository) Get(ctx context.Context, id string) (model.Friendship, error) {
	const query = `SELECT ` + friendshipColumns + ` FROM social.friendships WHERE id = $1::uuid`
	return scanFriendship(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresFriendshipRepository) GetByPair(ctx context.Context, a, b string) (model.Friendship, error) {
	const query = `
		SELECT ` + friendshipColumns + ` FROM social.friendships
		WHERE LEAST(requester_id, addressee_id) = LEAST($1::uuid, $2::uuid)
		  AND GREATEST(requester_id, addressee_id) = GREATEST($1::uuid, $2::uuid)`
	return scanFriendship(r.pool.QueryRow(ctx, query, a, b))
}

func (r *PostgresFriendshipRepository) Accept(ctx context.Context, id, addresseeID string) (model.Friendship, error) {
	const query = `
		UPDATE social.friendships
		SET status = 'accepted', accepted_at = now()
		WHERE id = $1::uuid AND addressee_id = $2::uuid AND status = 'pending'
		RETURNING ` + friendshipColumns
	return scanFriendship(r.pool.QueryRow(ctx, query, id, addresseeID))
}

func (r *PostgresFriendshipRepository) Delete(ctx context.Context, id string, status model.Status) error {
	const query = `DELETE FROM social.friendships WHERE id = $1::uuid AND status = $2`
	tag, err := r.pool.Exec(ctx, query, id, string(status))
	if err != nil {
		return mapFriendshipError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *PostgresFriendshipRepository) ListAccepted(ctx context.Context, profileID string) ([]model.Friendship, error) {
	const query = `
		SELECT ` + friendshipColumns + ` FROM social.friendships
		WHERE status = 'accepted' AND (requester_id = $1::uuid OR addressee_id = $1::uuid)
		ORDER BY accepted_at DESC`
	return r.list(ctx, query, profileID)
}

func (r *PostgresFriendshipRepository) ListPending(ctx context.Context, profileID string) ([]model.Friendship, error) {
	const query = `
		SELECT ` + friendshipColumns + ` FROM social.friendships
		WHERE status = 'pending' AND (requester_id = $1::uuid OR addressee_id = $1::uuid)
		ORDER BY created_at DESC`
	return r.list(ctx, query, profileID)
}

func (r *PostgresFriendshipRepository) list(ctx context.Context, query, profileID string) ([]model.Friendship, error) {
	rows, err := r.pool.Query(ctx, query, profileID)
	if err != nil {
		return nil, mapFriendshipError(err)
	}
	defer rows.Close()

	out := make([]model.Friendship, 0)
	for rows.Next() {
		f, err := scanFriendship(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, mapFriendshipError(rows.Err())
}
