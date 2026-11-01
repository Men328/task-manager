-- =============================================================================
-- 000014_task_recurrence (down)
-- =============================================================================

DROP INDEX IF EXISTS task.ix_tasks_recurrence_series;

ALTER TABLE task.TASKS
    DROP COLUMN IF EXISTS recurrence_next_id,
    DROP COLUMN IF EXISTS recurrence_series_id,
    DROP COLUMN IF EXISTS recurrence_tz,
    DROP COLUMN IF EXISTS recurrence_rule;
