package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/project/internal/model"
)

const projectColumns = `id::text, profile_id::text, name, description, COALESCE(color, ''), is_default, is_archived, created_at, updated_at`

type PostgresProjectRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresProjectRepository(pool *pgxpool.Pool) *PostgresProjectRepository {
	return &PostgresProjectRepository{pool: pool}
}

func mapProjectError(err error) error {
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
			switch pgErr.ConstraintName {
			case "uq_projects_profile_name":
				return model.NewError(model.ErrorKindNameDuplicate, "project name already exists")
			case "uq_projects_one_default":
				// Service đọc lại GetDefault khi gặp lỗi này lúc seed.
				return model.NewError(model.ErrorKindNameDuplicate, "default project already exists")
			}
		case "23503", "22P02":
			return model.ErrNotFound
		case "23514":
			if pgErr.ConstraintName == "ck_project_projects_color_format" {
				return model.NewError(model.ErrorKindColorInvalid, "project color is invalid")
			}
			return model.ErrInvalid
		case "22001":
			// Chỉ cột name có giới hạn độ dài mà người dùng nhập vào.
			return model.NewError(model.ErrorKindNameTooLong, "project name is too long")
		}
	}
	return err
}

func scanProject(row pgx.Row) (model.Project, error) {
	var p model.Project
	if err := row.Scan(&p.ID, &p.ProfileID, &p.Name, &p.Description, &p.Color,
		&p.IsDefault, &p.IsArchived, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return model.Project{}, mapProjectError(err)
	}
	return p, nil
}

func (r *PostgresProjectRepository) Create(ctx context.Context, p model.Project) (model.Project, error) {
	const query = `
		INSERT INTO project.projects (profile_id, name, description, color, is_default)
		VALUES ($1::uuid, $2, $3, NULLIF($4, ''), $5)
		RETURNING ` + projectColumns
	return scanProject(r.pool.QueryRow(ctx, query, p.ProfileID, p.Name, p.Description, p.Color, p.IsDefault))
}

func (r *PostgresProjectRepository) Get(ctx context.Context, id string) (model.Project, error) {
	const query = `SELECT ` + projectColumns + ` FROM project.projects WHERE id = $1::uuid`
	return scanProject(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresProjectRepository) GetDefault(ctx context.Context, profileID string) (model.Project, error) {
	const query = `SELECT ` + projectColumns + ` FROM project.projects WHERE profile_id = $1::uuid AND is_default`
	return scanProject(r.pool.QueryRow(ctx, query, profileID))
}

func (r *PostgresProjectRepository) List(ctx context.Context, f model.ProjectFilter) ([]model.Project, error) {
	const query = `
		SELECT ` + projectColumns + `
		FROM project.projects
		WHERE ($1::uuid IS NULL OR profile_id = $1::uuid)
		  AND ($2 OR NOT is_archived)
		ORDER BY is_default DESC, created_at`

	var profile any
	if f.ProfileID != "" {
		profile = f.ProfileID
	}
	rows, err := r.pool.Query(ctx, query, profile, f.IncludeArchived)
	if err != nil {
		return nil, mapProjectError(err)
	}
	defer rows.Close()

	out := make([]model.Project, 0)
	for rows.Next() {
		p, err := scanProject(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, mapProjectError(rows.Err())
}

func (r *PostgresProjectRepository) Update(ctx context.Context, id string, upd model.ProjectUpdate) (model.Project, error) {
	const query = `
		UPDATE project.projects
		SET name        = COALESCE($2, name),
		    description = COALESCE($3, description),
		    color       = COALESCE(NULLIF($4, ''), color)
		WHERE id = $1::uuid
		RETURNING ` + projectColumns
	return scanProject(r.pool.QueryRow(ctx, query, id, upd.Name, upd.Description, upd.Color))
}

func (r *PostgresProjectRepository) SetArchived(ctx context.Context, id string, archived bool) (model.Project, error) {
	const query = `UPDATE project.projects SET is_archived = $2 WHERE id = $1::uuid RETURNING ` + projectColumns
	return scanProject(r.pool.QueryRow(ctx, query, id, archived))
}
