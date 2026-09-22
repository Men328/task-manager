-- =============================================================================
-- 000003_init_workspace
-- workspace service: WORKSPACES (namespace gốc chứa mọi tài nguyên)
-- Phụ thuộc: 000001_init_identity (schema identity)
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS workspace;

-- ============================================================== WORKSPACES
CREATE TABLE workspace.WORKSPACES (
    id               uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_profile_id uuid         NOT NULL REFERENCES identity.PROFILES (id) ON DELETE CASCADE,
    name             varchar(120) NOT NULL,
    slug             varchar(120) NOT NULL,
    description      text,
    color            varchar(9),
    icon             varchar(64),
    is_default       boolean      NOT NULL DEFAULT false,
    position         integer      NOT NULL DEFAULT 0,
    is_archived      boolean      NOT NULL DEFAULT false,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    updated_at       timestamptz  NOT NULL DEFAULT now(),
    deleted_at       timestamptz,
    CONSTRAINT ck_workspaces_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT ck_workspaces_slug_format    CHECK (slug ~ '^[a-z0-9][a-z0-9_-]*$'),
    CONSTRAINT ck_workspaces_color_format   CHECK (color IS NULL OR color ~* '^#[0-9a-f]{6}([0-9a-f]{2})?$'),
    -- dành cho FK (owner_profile_id) về sau; xem design/db_schema.dbml
    CONSTRAINT uq_workspaces_id_owner       UNIQUE (id, owner_profile_id),
    CONSTRAINT uq_workspaces_owner_slug     UNIQUE (owner_profile_id, slug)
);

-- Mỗi owner chỉ có 1 workspace default đang hoạt động (DBML không diễn đạt được partial index)
CREATE UNIQUE INDEX IF NOT EXISTS uq_workspaces_one_default
    ON workspace.WORKSPACES (owner_profile_id)
    WHERE is_default AND NOT is_archived AND deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_workspaces_owner_position
    ON workspace.WORKSPACES (owner_profile_id, position);

CREATE INDEX IF NOT EXISTS ix_workspaces_owner_active
    ON workspace.WORKSPACES (owner_profile_id, is_archived, deleted_at);

-- ------------------------------------------------------------------ updated_at
CREATE OR REPLACE FUNCTION workspace.fn_set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_workspaces_updated_at ON workspace.WORKSPACES;
CREATE TRIGGER trg_workspaces_updated_at
    BEFORE UPDATE ON workspace.WORKSPACES
    FOR EACH ROW EXECUTE FUNCTION workspace.fn_set_updated_at();
