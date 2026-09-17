-- =============================================================================
-- 000001_init_identity
-- identity service: PROFILES + AUTH_PROVIDERS
-- Yêu cầu: PostgreSQL >= 13 (gen_random_uuid() có sẵn trong core từ PG13)
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS identity;

CREATE TABLE IF NOT EXISTS identity.PROFILES (
    id            uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    email         varchar(255) NOT NULL,
    password_hash varchar(255),
    display_name  varchar(100) NOT NULL,
    avatar_url    text,
    timezone      varchar(64)  NOT NULL DEFAULT 'Asia/Ho_Chi_Minh',
    locale        varchar(10)  NOT NULL DEFAULT 'vi',
    is_active     boolean      NOT NULL DEFAULT true,
    last_login_at timestamptz,
    created_at    timestamptz  NOT NULL DEFAULT now(),
    updated_at    timestamptz  NOT NULL DEFAULT now(),
    deleted_at    timestamptz,
    CONSTRAINT ck_profiles_email_not_blank        CHECK (btrim(email) <> ''),
    CONSTRAINT ck_profiles_display_name_not_blank CHECK (btrim(display_name) <> '')
);

-- email unique không phân biệt hoa/thường, chỉ áp dụng cho profile chưa xoá mềm
CREATE UNIQUE INDEX IF NOT EXISTS uq_profiles_email
    ON identity.PROFILES (lower(email))
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_profiles_active
    ON identity.PROFILES (is_active, deleted_at);

CREATE TABLE IF NOT EXISTS identity.AUTH_PROVIDERS (
    id               uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id       uuid         NOT NULL REFERENCES identity.PROFILES (id) ON DELETE CASCADE,
    provider         varchar(32)  NOT NULL,
    provider_user_id varchar(255) NOT NULL,
    created_at       timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT uq_auth_providers_profile_provider UNIQUE (profile_id, provider),
    CONSTRAINT uq_auth_providers_provider_uid     UNIQUE (provider, provider_user_id)
);

CREATE INDEX IF NOT EXISTS ix_auth_providers_profile
    ON identity.AUTH_PROVIDERS (profile_id);

-- ------------------------------------------------------------------ updated_at
CREATE OR REPLACE FUNCTION identity.fn_set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_profiles_updated_at ON identity.PROFILES;
CREATE TRIGGER trg_profiles_updated_at
    BEFORE UPDATE ON identity.PROFILES
    FOR EACH ROW EXECUTE FUNCTION identity.fn_set_updated_at();
