-- =============================================================================
-- 000007_init_event (down)
-- Gỡ toàn bộ schema event: bảng events + index + trigger function.
-- Đối xứng với up, không đụng tới schema khác (identity/task/calendar/...).
-- =============================================================================

DROP SCHEMA IF EXISTS event CASCADE;
