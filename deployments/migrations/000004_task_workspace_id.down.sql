-- =============================================================================
-- 000004_task_workspace_id (down)
-- Gỡ FK/index + cột workspace_id khỏi task.tasks.
-- Không xoá các workspace mặc định đã tạo ở bước up (tránh xoá nhầm dữ liệu).
-- =============================================================================

DROP INDEX IF EXISTS task.ix_tasks_workspace_active;
DROP INDEX IF EXISTS task.ix_tasks_workspace_due;
DROP INDEX IF EXISTS task.ix_tasks_workspace_parent_position;
DROP INDEX IF EXISTS task.ix_tasks_workspace_status;

ALTER TABLE task.tasks DROP CONSTRAINT IF EXISTS fk_tasks_workspace;

ALTER TABLE task.tasks DROP COLUMN IF EXISTS workspace_id;
