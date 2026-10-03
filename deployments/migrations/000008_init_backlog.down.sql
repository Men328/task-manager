-- =============================================================================
-- 000008_init_backlog (down)
-- Gỡ toàn bộ schema backlog: bảng backlogs + index + trigger function.
-- Đối xứng với up, không đụng tới schema khác (identity/task/calendar/event).
-- =============================================================================

DROP SCHEMA IF EXISTS backlog CASCADE;
