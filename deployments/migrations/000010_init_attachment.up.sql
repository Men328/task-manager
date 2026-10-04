-- =============================================================================
-- 000010_init_attachment
-- attachment service: tệp đính kèm cho task / schedule / event / backlog
-- Phụ thuộc: 000001_init_identity (identity.profiles)
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS attachment;

-- ============================================================= attachments
-- Một row = metadata của một tệp đính kèm. Liên kết đa hình tới đối tượng
-- nghiệp vụ qua (owner_type, owner_id) — không dùng FK vì 4 đối tượng nằm ở
-- 4 schema khác nhau. Nội dung nhị phân KHÔNG lưu ở DB mà nằm trên object
-- storage S3-compatible (RustFS) tại `object_key`.
CREATE TABLE attachment.attachments (
    id            uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id    uuid         NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    owner_type    varchar(20)  NOT NULL,
    owner_id      uuid         NOT NULL,
    file_name     varchar(255) NOT NULL,
    content_type  varchar(255) NOT NULL DEFAULT 'application/octet-stream',
    size          bigint       NOT NULL,
    object_key    varchar(1024) NOT NULL,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_attachment_attachments_owner_type
        CHECK (owner_type IN ('task', 'schedule', 'event', 'backlog')),
    CONSTRAINT ck_attachment_attachments_file_name_not_blank
        CHECK (btrim(file_name) <> ''),
    CONSTRAINT ck_attachment_attachments_object_key_not_blank
        CHECK (btrim(object_key) <> ''),
    CONSTRAINT ck_attachment_attachments_size
        CHECK (size >= 0)
);

-- Truy vấn chính: tệp của 1 đối tượng, mới nhất trước.
CREATE INDEX IF NOT EXISTS ix_attachment_attachments_owner
    ON attachment.attachments (owner_type, owner_id, created_at DESC);

-- Tệp của 1 profile (dọn dẹp / kiểm tra dung lượng).
CREATE INDEX IF NOT EXISTS ix_attachment_attachments_profile
    ON attachment.attachments (profile_id);
