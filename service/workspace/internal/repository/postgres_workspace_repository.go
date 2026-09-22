package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/workspace/internal/model"
)

const workspaceColumns = `id::text, owner_profile_id::text, name, slug, COALESCE(description, ''), COALESCE(color, ''), COALESCE(icon, ''), is_default, position, is_archived, created_at, updated_at, deleted_at`

type PostgresWorkspaceRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresWorkspaceRepository(pool *pgxpool.Pool) *PostgresWorkspaceRepository {
	return &PostgresWorkspaceRepository{pool: pool}
}

func mapWorkspaceError(err error) error {
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
				return model.NewError(model.ErrorKindSlugAlreadyExists,
					"slug %s đã tồn tại trong workspace của profile", pgErr.ConstraintName)
			}
			return model.ErrAlreadyExists
		case "23503", "22P02":
			return model.ErrNotFound
		}
	}
	return err
}

func scanWorkspace(row pgx.Row) (model.Workspace, error) {
	var w model.Workspace
	if err := row.Scan(
		&w.ID,
		&w.OwnerProfileID,
		&w.Name,
		&w.Slug,
		&w.Description,
		&w.Color,
		&w.Icon,
		&w.IsDefault,
		&w.Position,
		&w.IsArchived,
		&w.CreatedAt,
		&w.UpdatedAt,
		&w.DeletedAt,
	); err != nil {
		return model.Workspace{}, mapWorkspaceError(err)
	}
	return w, nil
}

func (r *PostgresWorkspaceRepository) Create(ctx context.Context, w model.Workspace) (model.Workspace, error) {
	const clearDefault = `
		UPDATE workspace.workspaces
		SET is_default = false
		WHERE owner_profile_id = $1::uuid
		  AND is_default
		  AND NOT is_archived
		  AND deleted_at IS NULL`

	const insert = `
		INSERT INTO workspace.workspaces
			(id, owner_profile_id, name, slug, description, color, icon, is_default, position)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid, $3, $4,
			 NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), $8, $9)
		RETURNING ` + workspaceColumns

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Workspace{}, mapWorkspaceError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if w.IsDefault {
		if _, err := tx.Exec(ctx, clearDefault, w.OwnerProfileID); err != nil {
			return model.Workspace{}, mapWorkspaceError(err)
		}
	}

	created, err := scanWorkspace(tx.QueryRow(ctx, insert,
		w.ID, w.OwnerProfileID, w.Name, w.Slug, w.Description, w.Color, w.Icon, w.IsDefault, w.Position))
	if err != nil {
		return model.Workspace{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Workspace{}, mapWorkspaceError(err)
	}
	return created, nil
}

func (r *PostgresWorkspaceRepository) Get(ctx context.Context, id string) (model.Workspace, error) {
	const query = `SELECT ` + workspaceColumns + ` FROM workspace.workspaces WHERE id = $1::uuid AND deleted_at IS NULL`
	return scanWorkspace(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresWorkspaceRepository) List(ctx context.Context, f model.WorkspaceFilter) ([]model.Workspace, error) {
	const query = `
		SELECT ` + workspaceColumns + `
		FROM workspace.workspaces
		WHERE deleted_at IS NULL
		  AND ($1::uuid IS NULL OR owner_profile_id = $1::uuid)
		  AND ($2 OR NOT is_archived)
		ORDER BY position, created_at`

	var owner any
	if f.OwnerProfileID != "" {
		owner = f.OwnerProfileID
	}

	rows, err := r.pool.Query(ctx, query, owner, f.IncludeArchived)
	if err != nil {
		return nil, mapWorkspaceError(err)
	}
	defer rows.Close()

	out := make([]model.Workspace, 0)
	for rows.Next() {
		w, err := scanWorkspace(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, mapWorkspaceError(rows.Err())
}

func (r *PostgresWorkspaceRepository) Update(ctx context.Context, id string, upd model.WorkspaceUpdate) (model.Workspace, error) {
	const clearDefault = `
		UPDATE workspace.workspaces
		SET is_default = false
		WHERE owner_profile_id = (SELECT owner_profile_id FROM workspace.workspaces WHERE id = $1::uuid)
		  AND id <> $1::uuid
		  AND is_default
		  AND NOT is_archived
		  AND deleted_at IS NULL`

	const update = `
		UPDATE workspace.workspaces
		SET name        = COALESCE($2, name),
		    description = COALESCE($3, description),
		    color       = COALESCE($4, color),
		    icon        = COALESCE($5, icon),
		    position    = COALESCE($6, position),
		    is_archived = COALESCE($7, is_archived),
		    is_default  = COALESCE($8, is_default)
		WHERE id = $1::uuid AND deleted_at IS NULL
		RETURNING ` + workspaceColumns

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return model.Workspace{}, mapWorkspaceError(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if upd.IsDefault != nil && *upd.IsDefault {
		if _, err := tx.Exec(ctx, clearDefault, id); err != nil {
			return model.Workspace{}, mapWorkspaceError(err)
		}
	}

	updated, err := scanWorkspace(tx.QueryRow(ctx, update, id,
		upd.Name, upd.Description, upd.Color, upd.Icon, upd.Position, upd.IsArchived, upd.IsDefault))
	if err != nil {
		return model.Workspace{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return model.Workspace{}, mapWorkspaceError(err)
	}
	return updated, nil
}
