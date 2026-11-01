-- =============================================================================
-- 000014_task_recurrence
-- Task lặp lại: khi task có quy tắc lặp được hoàn thành, task-grpc tạo task kế
-- tiếp với hạn mới. recurrence_next_id chống sinh trùng khi mở lại rồi xong lại.
-- Phụ thuộc: 000002_init_task, 000013_task_reminders
-- =============================================================================

ALTER TABLE task.TASKS
    ADD COLUMN IF NOT EXISTS recurrence_rule      text,
    ADD COLUMN IF NOT EXISTS recurrence_tz        varchar(64),
    ADD COLUMN IF NOT EXISTS recurrence_series_id uuid,
    ADD COLUMN IF NOT EXISTS recurrence_next_id   uuid;

CREATE INDEX IF NOT EXISTS ix_tasks_recurrence_series
    ON task.TASKS (recurrence_series_id)
    WHERE recurrence_series_id IS NOT NULL;
