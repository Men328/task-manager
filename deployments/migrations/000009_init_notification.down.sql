-- =============================================================================
-- 000009_init_notification (down)
-- Gỡ toàn bộ schema notification: bảng notices + index + trigger function.
-- Đối xứng với up, không đụng tới schema khác.
-- =============================================================================

DROP SCHEMA IF EXISTS notification CASCADE;
