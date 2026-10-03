-- =============================================================================
-- 000006_init_calendar (down)
-- Gỡ toàn bộ schema calendar: bảng schedules + index + trigger function.
-- Đối xứng với up, không đụng tới schema khác (identity/task/mail_provider).
-- =============================================================================

DROP SCHEMA IF EXISTS calendar CASCADE;
