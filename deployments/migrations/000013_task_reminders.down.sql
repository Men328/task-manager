-- =============================================================================
-- 000013_task_reminders (down)
-- =============================================================================

DROP INDEX IF EXISTS task.ix_tasks_reminder_due;

ALTER TABLE task.TASKS
    DROP COLUMN IF EXISTS reminded_at,
    DROP COLUMN IF EXISTS remind_at;
