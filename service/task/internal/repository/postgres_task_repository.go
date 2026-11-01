package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"taskmanager/service/task/internal/model"
)

const taskColumns = `id::text, profile_id::text, project_id::text, COALESCE(parent_task_id::text, ''), status_id::text, title, COALESCE(description, ''), priority::text, position, start_at, due_at, completed_at, is_archived, created_at, updated_at, remind_at, reminded_at`

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
		&t.ProjectID,
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
		&t.RemindAt,
		&t.RemindedAt,
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
			(id, profile_id, parent_task_id, status_id, title, description,
			 priority, position, start_at, due_at, completed_at, is_archived, project_id,
			 remind_at, reminded_at)
		VALUES
			(COALESCE(NULLIF($1, '')::uuid, gen_random_uuid()), $2::uuid,
			 NULLIF($3, '')::uuid, $4::uuid, $5, NULLIF($6, ''), $7::task.task_priority,
			 $8, $9, $10, $11, $12, $13::uuid, $14, $15)
		RETURNING ` + taskColumns

	return scanTask(r.pool.QueryRow(ctx, query,
		t.ID, t.ProfileID, t.ParentTaskID, t.StatusID, t.Title, t.Description,
		priorityToDB(t.Priority), t.Position, t.StartAt, t.DueAt, t.CompletedAt, t.IsArchived, t.ProjectID,
		t.RemindAt, t.RemindedAt))
}

func (r *PostgresTaskRepository) Get(ctx context.Context, id string) (model.Task, error) {
	const query = `SELECT ` + taskColumns + ` FROM task.tasks WHERE id = $1::uuid AND deleted_at IS NULL`
	return scanTask(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresTaskRepository) List(ctx context.Context, f model.TaskFilter) ([]model.Task, error) {
	query, args := buildTaskListQuery(f)
	rows, err := r.pool.Query(ctx, query, args...)
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
		    is_archived    = COALESCE($12, is_archived),
		    remind_at      = CASE WHEN $13 THEN $14::timestamptz ELSE remind_at END,
		    reminded_at    = CASE WHEN $13 THEN $15::timestamptz ELSE reminded_at END
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
		upd.ChangesReminder(), upd.RemindAt, upd.RemindedAt,
	))
}

func (r *PostgresTaskRepository) ClaimDueReminders(ctx context.Context, limit int) ([]model.Task, error) {
	const query = `
		UPDATE task.tasks
		SET reminded_at = now()
		WHERE id IN (
			SELECT t.id
			FROM task.tasks t
			JOIN task.task_statuses s ON s.id = t.status_id
			WHERE t.remind_at <= now() AND t.reminded_at IS NULL
			  AND t.deleted_at IS NULL AND NOT t.is_archived
			  AND s.category NOT IN ('done', 'cancelled')
			ORDER BY t.remind_at
			LIMIT $1
			FOR UPDATE OF t SKIP LOCKED)
		RETURNING ` + taskColumns

	rows, err := r.pool.Query(ctx, query, limit)
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

func (r *PostgresTaskRepository) ReleaseReminder(ctx context.Context, id string, claimedAt time.Time) error {
	const query = `UPDATE task.tasks SET reminded_at = NULL WHERE id = $1::uuid AND reminded_at = $2`
	_, err := r.pool.Exec(ctx, query, id, claimedAt)
	return mapTaskError(err)
}

// SetProjectForSubtree đổi project của task rootID và mọi hậu duệ (theo
// parent_task_id) trong một câu UPDATE, giới hạn trong profile của task gốc.
func (r *PostgresTaskRepository) SetProjectForSubtree(ctx context.Context, rootID, projectID string) error {
	const query = `
		WITH RECURSIVE subtree AS (
			SELECT id, profile_id
			FROM task.tasks
			WHERE id = $1::uuid AND deleted_at IS NULL
			UNION
			SELECT c.id, c.profile_id
			FROM task.tasks c
			JOIN subtree s ON c.parent_task_id = s.id AND c.profile_id = s.profile_id
			WHERE c.deleted_at IS NULL
		)
		UPDATE task.tasks
		SET project_id = $2::uuid
		WHERE id IN (SELECT id FROM subtree)`

	tag, err := r.pool.Exec(ctx, query, rootID, projectID)
	if err != nil {
		return mapTaskError(err)
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound
	}
	return nil
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

const taskSearchExpr = `task.f_search_text(title || ' ' || COALESCE(description, ''))`

var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

func escapeLike(value string) string {
	return likeEscaper.Replace(value)
}

func buildTaskListQuery(f model.TaskFilter) (string, []any) {
	conds := []string{"deleted_at IS NULL"}
	args := make([]any, 0, 12)
	where := func(cond string, value any) {
		args = append(args, value)
		conds = append(conds, strings.ReplaceAll(cond, "?", fmt.Sprintf("$%d", len(args))))
	}

	if f.ProfileID != "" {
		where("profile_id = ?::uuid", f.ProfileID)
	}
	if f.StatusID != "" {
		where("status_id = ?::uuid", f.StatusID)
	}
	if f.RootOnly {
		conds = append(conds, "parent_task_id IS NULL")
	} else if f.ParentTaskID != "" {
		where("parent_task_id = ?::uuid", f.ParentTaskID)
	}
	if !f.IncludeArchived {
		conds = append(conds, "NOT is_archived")
	}
	if f.ProjectID != "" {
		where("project_id = ?::uuid", f.ProjectID)
	}
	if f.DueFrom != nil {
		where("due_at >= ?", *f.DueFrom)
	}
	if f.DueTo != nil {
		where("due_at < ?", *f.DueTo)
	}
	if f.HasDueAt != nil {
		if *f.HasDueAt {
			conds = append(conds, "due_at IS NOT NULL")
		} else {
			conds = append(conds, "due_at IS NULL")
		}
	}
	if len(f.Priorities) > 0 {
		values := make([]string, 0, len(f.Priorities))
		for _, p := range f.Priorities {
			values = append(values, priorityToDB(p))
		}
		where("priority = ANY(?::task.task_priority[])", values)
	}
	if len(f.ExcludeStatusIDs) > 0 {
		where("NOT (status_id = ANY(?::uuid[]))", f.ExcludeStatusIDs)
	}
	orderBy := taskOrderBy(f.SortField, f.SortDesc)
	if f.Query != "" {
		where(taskSearchExpr+` LIKE '%' || task.f_search_text(?) || '%' ESCAPE '\'`, escapeLike(f.Query))
		if f.SortField == model.TaskSortUnspecified {
			orderBy = fmt.Sprintf("similarity(%s, task.f_search_text($%d)) DESC, tasks.updated_at DESC, tasks.id", taskSearchExpr, len(args))
		}
	}

	query := `SELECT ` + taskColumns + ` FROM task.tasks WHERE ` + strings.Join(conds, " AND ") +
		` ORDER BY ` + orderBy
	if f.Limit > 0 {
		args = append(args, f.Limit, f.Offset)
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	}
	return query, args
}

func taskOrderBy(field model.TaskSortField, desc bool) string {
	dir := "ASC"
	if desc {
		dir = "DESC"
	}
	switch field {
	case model.TaskSortDueAt:
		return "tasks.due_at " + dir + " NULLS LAST, tasks.id"
	case model.TaskSortPriority:
		return "tasks.priority " + dir + ", tasks.id"
	case model.TaskSortCreatedAt:
		return "tasks.created_at " + dir + ", tasks.id"
	case model.TaskSortUpdatedAt:
		return "tasks.updated_at " + dir + ", tasks.id"
	default:
		return "tasks.position " + dir + ", tasks.created_at " + dir + ", tasks.id"
	}
}
