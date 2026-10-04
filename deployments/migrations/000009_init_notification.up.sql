-- =============================================================================
-- 000009_init_notification
-- notification service: notices (thông báo của user, đẩy qua soketi)
-- Phụ thuộc: 000001_init_identity (identity.profiles)
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS notification;

-- =================================================================== notices
-- Mỗi dòng là 1 thông báo thuộc 1 profile. mail-provider tạo notice sau khi
-- phân loại email thành công; notification service lưu rồi push qua soketi vào
-- kênh riêng của user (private-noti-internal-<profile_id>).
-- target_type + target_id trỏ tới đối tượng nguồn để UI mở đúng trang.
CREATE TABLE IF NOT EXISTS notification.notices (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid         NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    -- Nhãn nghiệp vụ: TASK_CREATED | SCHEDULE_CREATED | EVENT_CREATED | BACKLOG_CREATED.
    type        varchar(64)  NOT NULL,
    title       varchar(500) NOT NULL,
    body        text,
    -- task | schedule | event | backlog.
    target_type varchar(32)  NOT NULL,
    target_id   uuid         NOT NULL,
    -- Gmail message id nguồn (nếu notice sinh ra từ email).
    source      varchar(255),
    is_read     boolean      NOT NULL DEFAULT false,
    read_at     timestamptz,
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_notification_notices_title_not_blank
        CHECK (btrim(title) <> ''),
    CONSTRAINT ck_notification_notices_target_type
        CHECK (target_type IN ('task', 'schedule', 'event', 'backlog')),
    CONSTRAINT ck_notification_notices_read_at
        CHECK (is_read OR read_at IS NULL)
);

-- Truy vấn chính: đếm/list notice CHƯA đọc của 1 profile.
CREATE INDEX IF NOT EXISTS ix_notification_notices_profile_unread
    ON notification.notices (profile_id, created_at DESC)
    WHERE is_read = false;

-- Truy vấn danh sách notice mới nhất của 1 profile (mọi trạng thái).
CREATE INDEX IF NOT EXISTS ix_notification_notices_profile_created
    ON notification.notices (profile_id, created_at DESC);

-- Truy vết đối tượng nguồn.
CREATE INDEX IF NOT EXISTS ix_notification_notices_target
    ON notification.notices (target_type, target_id);

-- ------------------------------------------------------------------ updated_at
CREATE OR REPLACE FUNCTION notification.fn_set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_notification_notices_updated_at ON notification.notices;
CREATE TRIGGER trg_notification_notices_updated_at
    BEFORE UPDATE ON notification.notices
    FOR EACH ROW EXECUTE FUNCTION notification.fn_set_updated_at();
