-- =============================================================================
-- 000011_init_project (down)
-- =============================================================================

DROP INDEX IF EXISTS event.ix_event_events_profile_project;
DROP INDEX IF EXISTS calendar.ix_calendar_schedules_profile_project;
DROP INDEX IF EXISTS task.ix_tasks_profile_project;

ALTER TABLE event.events        DROP CONSTRAINT IF EXISTS fk_event_events_project;
ALTER TABLE calendar.schedules  DROP CONSTRAINT IF EXISTS fk_calendar_schedules_project;
ALTER TABLE task.TASKS          DROP CONSTRAINT IF EXISTS fk_tasks_project;

ALTER TABLE event.events        DROP COLUMN IF EXISTS project_id;
ALTER TABLE calendar.schedules  DROP COLUMN IF EXISTS project_id;
ALTER TABLE task.TASKS          DROP COLUMN IF EXISTS project_id;

DROP SCHEMA IF EXISTS project CASCADE;
