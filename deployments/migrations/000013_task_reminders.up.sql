-- =============================================================================
-- 000013_task_reminders
-- Nhắc việc: mỗi task có tối đa 1 thời điểm nhắc. Worker trong task-grpc claim
-- các nhắc đến hạn (reminded_at IS NULL) rồi gửi notice qua notification service.
-- Phụ thuộc: 000002_init_task
-- =============================================================================

ALTER TABLE task.TASKS
    ADD COLUMN IF NOT EXISTS remind_at   timestamptz,
    ADD COLUMN IF NOT EXISTS reminded_at timestamptz;

-- Chỉ chứa nhắc còn chờ gửi -> worker quét theo remind_at rất rẻ.
CREATE INDEX IF NOT EXISTS ix_tasks_reminder_due
    ON task.TASKS (remind_at)
    WHERE remind_at IS NOT NULL AND reminded_at IS NULL
      AND deleted_at IS NULL AND NOT is_archived;
