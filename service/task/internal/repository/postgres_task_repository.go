package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/task/internal/model"
)

const taskColumns = `id::text, profile_id::text, workspace_id::text, COALESCE(parent_task_id::text, ''), status_id::text, title, COALESCE(description, ''), priority::text, position, start_at, due_at, completed_at, is_archived, created_at, updated_at`

const statusLogColumns = `id, task_id::text, profile_id::text, COALESCE(from_status_id::text, ''), to_status_id::text, COALESCE(note, ''), changed_at`

type PostgresTaskRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresTaskRepository(pool *pgxpool.Pool) *PostgresTaskRepository {
	return &PostgresTaskRepository{pool: pool}
}

func mapTaskError(err error) error {
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
		case "23514":
			return model.ErrCycle
		}
	}
	return err
}

func scanTask(row pgx.Row) (model.Task, error) {
	var t model.Task
	var priority string
	if err := row.Scan(
		&t.ID,
		&t.ProfileID,
		&t.WorkspaceID,
		&t.ParentTaskID,
		&t.StatusID,
		&t.Title,
		&t.Description,
		&priority,
		&t.Position,
		&t.StartAt,
		&t.DueAt,
		&t.CompletedAt,
		&t.IsArchived,
		&t.CreatedAt,
		&t.UpdatedAt,
	); err != nil {
		return model.Task{}, mapTaskError(err)
	}
	t.Priority = priorityFromDB(priority)
	return t, nil
}

func scanStatusLog(row pgx.Row) (model.StatusLog, error) {
	var log model.StatusLog
	if err := row.Scan(
		&log.ID,
		&log.TaskID,
		&log.ProfileID,
		&log.FromStatusID,
		&log.ToStatusID,
		&log.Note,
		&log.ChangedAt,
	); err != nil {
		return model.StatusLog{}, mapTaskError(err)
	}
	return log, nil
}

func (r *PostgresTaskRepository) Create(ctx context.Context, t model.Task) (model.Task, error) {
	const query = `
		INSERT INTO task.tasks
			(id, profile_id, workspace_id, parent_task_id, status_id, title, description,
			 priority, position, start_at, due_at, completed_at, is_archived)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid, $3::uuid,
			 NULLIF($4, '')::uuid, $5::uuid, $6, NULLIF($7, ''), $8::task.task_priority,
			 $9, $10, $11, $12, $13)
		RETURNING ` + taskColumns

	return scanTask(r.pool.QueryRow(ctx, query,
		t.ID, t.ProfileID, t.WorkspaceID, t.ParentTaskID, t.StatusID, t.Title, t.Description,
		priorityToDB(t.Priority), t.Position, t.StartAt, t.DueAt, t.CompletedAt, t.IsArchived))
}

func (r *PostgresTaskRepository) Get(ctx context.Context, id string) (model.Task, error) {
	const query = `SELECT ` + taskColumns + ` FROM task.tasks WHERE id = $1::uuid AND deleted_at IS NULL`
	return scanTask(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresTaskRepository) List(ctx context.Context, f model.TaskFilter) ([]model.Task, error) {
	const query = `
		SELECT ` + taskColumns + `
		FROM task.tasks
		WHERE deleted_at IS NULL
		  AND ($1::uuid IS NULL OR profile_id = $1::uuid)
		  AND ($2::uuid IS NULL OR workspace_id = $2::uuid)
		  AND ($3::uuid IS NULL OR status_id = $3::uuid)
		  AND (CASE WHEN $4 THEN parent_task_id IS NULL
		            ELSE ($5::uuid IS NULL OR parent_task_id = $5::uuid) END)
		  AND ($6 OR NOT is_archived)
		ORDER BY position, created_at`

	rows, err := r.pool.Query(ctx, query,
		optionalUUID(f.ProfileID),
		optionalUUID(f.WorkspaceID),
		optionalUUID(f.StatusID),
		f.RootOnly,
		optionalUUID(f.ParentTaskID),
		f.IncludeArchived,
	)
	if err != nil {
		return nil, mapTaskError(err)
	}
	defer rows.Close()

	out := make([]model.Task, 0)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, mapTaskError(rows.Err())
}

func (r *PostgresTaskRepository) ListChildren(ctx context.Context, parentTaskID string) ([]model.Task, error) {
	const query = `
		SELECT ` + taskColumns + `
		FROM task.tasks
		WHERE parent_task_id = $1::uuid AND deleted_at IS NULL
		ORDER BY position, created_at`

	rows, err := r.pool.Query(ctx, query, parentTaskID)
	if err != nil {
		return nil, mapTaskError(err)
	}
	defer rows.Close()

	out := make([]model.Task, 0)
	for rows.Next() {
		t, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, mapTaskError(rows.Err())
}

func (r *PostgresTaskRepository) Update(ctx context.Context, id string, upd model.TaskUpdate) (model.Task, error) {
	const query = `
		UPDATE task.tasks
		SET parent_task_id = CASE WHEN $2 THEN NULLIF($3, '')::uuid ELSE parent_task_id END,
		    title          = COALESCE($4, title),
		    description    = COALESCE($5, description),
		    priority       = COALESCE($6::task.task_priority, priority),
		    position       = COALESCE($7, position),
		    start_at       = CASE WHEN $8 THEN $9::timestamptz ELSE start_at END,
		    due_at         = CASE WHEN $10 THEN $11::timestamptz ELSE due_at END,
		    is_archived    = COALESCE($12, is_archived)
		WHERE id = $1::uuid AND deleted_at IS NULL
		RETURNING ` + taskColumns

	var parentValue string
	if upd.ParentTaskID != nil {
		parentValue = *upd.ParentTaskID
	}

	var startAt, dueAt any
	if upd.StartAt != nil {
		startAt = *upd.StartAt
	}
	if upd.DueAt != nil {
		dueAt = *upd.DueAt
	}

	return scanTask(r.pool.QueryRow(ctx, query,
		id,
		upd.ParentTaskID != nil, parentValue,
		upd.Title,
		upd.Description,
		priorityToDBPtr(upd.Priority),
		upd.Position,
		upd.StartAt != nil, startAt,
		upd.DueAt != nil, dueAt,
		upd.IsArchived,
	))
}

func (r *PostgresTaskRepository) Delete(ctx context.Context, id string) error {
	const query = `DELETE FROM task.tasks WHERE id = $1::uuid`

	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return mapTaskError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
}

func (r *PostgresTaskRepository) ChangeStatus(ctx context.Context, id, statusID string, completedAt *time.Time) (model.Task, error) {
	const query = `
		UPDATE task.tasks
		SET status_id = $2::uuid,
		    completed_at = $3
		WHERE id = $1::uuid AND deleted_at IS NULL
		RETURNING ` + taskColumns

	return scanTask(r.pool.QueryRow(ctx, query, id, statusID, completedAt))
}

func (r *PostgresTaskRepository) CountByStatus(ctx context.Context, statusID string) (int, error) {
	const query = `SELECT count(*) FROM task.tasks WHERE status_id = $1::uuid AND deleted_at IS NULL`

	var count int
	if err := r.pool.QueryRow(ctx, query, statusID).Scan(&count); err != nil {
		return 0, mapTaskError(err)
	}
	return count, nil
}

func (r *PostgresTaskRepository) AppendStatusLog(ctx context.Context, log model.StatusLog) error {
	const query = `
		INSERT INTO task.task_status_logs
			(task_id, profile_id, from_status_id, to_status_id, note, changed_at)
		VALUES
			($1::uuid, $2::uuid, NULLIF($3, '')::uuid, $4::uuid, NULLIF($5, ''), COALESCE($6::timestamptz, now()))`

	var changedAt any
	if !log.ChangedAt.IsZero() {
		changedAt = log.ChangedAt
	}

	_, err := r.pool.Exec(ctx, query,
		log.TaskID, log.ProfileID, log.FromStatusID, log.ToStatusID, log.Note, changedAt)
	return mapTaskError(err)
}

func (r *PostgresTaskRepository) ListStatusLogs(ctx context.Context, taskID string) ([]model.StatusLog, error) {
	const query = `
		SELECT ` + statusLogColumns + `
		FROM task.task_status_logs
		WHERE task_id = $1::uuid
		ORDER BY changed_at`

	rows, err := r.pool.Query(ctx, query, taskID)
	if err != nil {
		return nil, mapTaskError(err)
	}
	defer rows.Close()

	out := make([]model.StatusLog, 0)
	for rows.Next() {
		log, err := scanStatusLog(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, log)
	}
	return out, mapTaskError(rows.Err())
}
