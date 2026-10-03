-- =============================================================================
-- 000008_init_backlog
-- backlog service: backlogs (dữ liệu mail worker không phân loại được)
-- Phụ thuộc: 000001_init_identity (identity.profiles)
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS backlog;

-- ================================================================== backlogs
-- Mail worker đẩy mọi email KHÔNG thuộc task/schedule/event vào đây
-- (category = 'other'), kèm lý do và object key của email gốc trên object storage.
-- Người dùng cũng có thể tự tạo mục backlog trên UI.
CREATE TABLE backlog.backlogs (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid         NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    title       varchar(500) NOT NULL,
    description text,
    -- Header From của email nguồn (rỗng/NULL nếu người dùng tự tạo).
    sender      varchar(320),
    -- Gmail message id; dùng để truy vết / chống trùng.
    source      varchar(255),
    -- Nhãn phân loại thô của mail worker: task | schedule | event | other.
    category    varchar(20)  NOT NULL DEFAULT 'other',
    -- Lý do bị đẩy vào backlog.
    reason      text,
    -- Object key trên object storage S3 (email gốc dạng JSON).
    object_key  varchar(512),
    status      varchar(20)  NOT NULL DEFAULT 'NEW',
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_backlog_backlogs_title_not_blank
        CHECK (btrim(title) <> ''),
    CONSTRAINT ck_backlog_backlogs_status
        CHECK (status IN ('NEW', 'TRIAGED', 'ARCHIVED'))
);

-- Truy vấn chính: backlog mới nhất của 1 profile.
CREATE INDEX IF NOT EXISTS ix_backlog_backlogs_profile_created
    ON backlog.backlogs (profile_id, created_at DESC);

-- Truy vết / chống trùng theo message id nguồn.
CREATE INDEX IF NOT EXISTS ix_backlog_backlogs_source
    ON backlog.backlogs (source)
    WHERE source IS NOT NULL;

-- ------------------------------------------------------------------ updated_at
CREATE OR REPLACE FUNCTION backlog.fn_set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_backlog_backlogs_updated_at ON backlog.backlogs;
CREATE TRIGGER trg_backlog_backlogs_updated_at
    BEFORE UPDATE ON backlog.backlogs
    FOR EACH ROW EXECUTE FUNCTION backlog.fn_set_updated_at();
