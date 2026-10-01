-- =============================================================================
-- 000005_init_mail_provider (down)
-- Gỡ toàn bộ schema mail_provider: 2 bảng + index + trigger function.
-- Đối xứng với up, không đụng tới schema khác (identity/task).
-- =============================================================================

DROP SCHEMA IF EXISTS mail_provider CASCADE;
