-- =============================================================================
-- 000004_task_workspace_id
-- Thêm task.tasks.workspace_id (namespace gốc) + backfill workspace mặc định.
-- Phụ thuộc: 000002_init_task, 000003_init_workspace
-- Lưu ý: cột NOT NULL, phải chạy migration + deploy binary task mới cùng lúc.
-- =============================================================================

ALTER TABLE task.tasks ADD COLUMN workspace_id uuid;

-- Mỗi profile đang có task nhưng chưa có workspace -> tạo 1 workspace mặc định.
INSERT INTO workspace.workspaces (owner_profile_id, name, slug, is_default)
SELECT DISTINCT t.profile_id, 'Công việc của tôi', 'default', true
FROM task.tasks t
WHERE NOT EXISTS (
    SELECT 1
    FROM workspace.workspaces w
    WHERE w.owner_profile_id = t.profile_id
      AND w.deleted_at IS NULL
)
ON CONFLICT (owner_profile_id, slug) DO NOTHING;

-- Ưu tiên gán task về workspace default đang hoạt động.
UPDATE task.tasks t
SET workspace_id = w.id
FROM workspace.workspaces w
WHERE w.owner_profile_id = t.profile_id
  AND w.deleted_at IS NULL
  AND w.is_default
  AND t.workspace_id IS NULL;

-- Phần còn lại gán về workspace hoạt động bất kỳ của profile.
UPDATE task.tasks t
SET workspace_id = w.id
FROM workspace.workspaces w
WHERE w.owner_profile_id = t.profile_id
  AND w.deleted_at IS NULL
  AND t.workspace_id IS NULL;

ALTER TABLE task.tasks ALTER COLUMN workspace_id SET NOT NULL;

ALTER TABLE task.tasks
    ADD CONSTRAINT fk_tasks_workspace FOREIGN KEY (workspace_id)
    REFERENCES workspace.workspaces (id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS ix_tasks_workspace_status
    ON task.tasks (workspace_id, status_id);
CREATE INDEX IF NOT EXISTS ix_tasks_workspace_parent_position
    ON task.tasks (workspace_id, parent_task_id, position);
CREATE INDEX IF NOT EXISTS ix_tasks_workspace_due
    ON task.tasks (workspace_id, due_at);
CREATE INDEX IF NOT EXISTS ix_tasks_workspace_active
    ON task.tasks (workspace_id, is_archived, deleted_at);
