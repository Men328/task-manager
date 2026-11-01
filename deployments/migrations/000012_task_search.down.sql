-- =============================================================================
-- 000012_task_search (down)
-- Giữ lại extension pg_trgm/unaccent: có thể đã được dùng ở nơi khác.
-- =============================================================================

DROP INDEX IF EXISTS task.ix_tasks_search_trgm;
DROP FUNCTION IF EXISTS task.f_search_text(text);
