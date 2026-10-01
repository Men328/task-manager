-- =============================================================================
-- 000005_init_mail_provider
-- mail_provider service: sessions (kết nối Gmail: refresh token của user) +
--                        noti_indexes (checkpoint historyId + hạn watch)
-- Phụ thuộc: 000001_init_identity (identity.profiles)
-- Lưu ý: migration này CHỈ tạo bảng. Service mail-provider hiện vẫn dùng store
-- in-memory; 2 bảng này được dùng khi repository Postgres được bật lên.
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS mail_provider;

-- ==================================================================== sessions
-- Một row = một hộp thư đã uỷ quyền cho app. Chứa credential -> không SELECT
-- refresh_token ra ngoài tầng repository, và phải mã hoá at-rest.
-- refresh_token NULL = Google không trả refresh token (watch khi đó chỉ sống ~1h).
CREATE TABLE mail_provider.sessions (
    id                      uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id              uuid         NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    provider                varchar(32)  NOT NULL DEFAULT 'google',
    email                   varchar(255) NOT NULL,
    refresh_token           text,
    access_token            text,
    access_token_expires_at timestamptz,
    revoked_at              timestamptz,
    created_at              timestamptz  NOT NULL DEFAULT now(),
    updated_at              timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_mail_provider_sessions_email_not_blank
        CHECK (btrim(email) <> ''),
    CONSTRAINT ck_mail_provider_sessions_provider_not_blank
        CHECK (btrim(provider) <> ''),
    -- 1 profile chỉ có 1 kết nối cho mỗi provider; login lại = UPDATE row cũ.
    CONSTRAINT uq_mail_provider_sessions_profile_provider
        UNIQUE (profile_id, provider)
);

-- Notice Pub/Sub chỉ mang emailAddress -> đây là khoá tra ngược ra profile_id
-- (thay cho index byEmail đang nằm trong RAM).
-- Partial + lower(): hộp thư đã ngắt kết nối (revoked_at) không chặn hộp thư mới,
-- và email không phân biệt hoa/thường (giống uq_profiles_email).
CREATE UNIQUE INDEX IF NOT EXISTS uq_mail_provider_sessions_email
    ON mail_provider.sessions (lower(email))
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS ix_mail_provider_sessions_profile
    ON mail_provider.sessions (profile_id);

-- ================================================================ noti_indexes
-- Checkpoint historyId: con trỏ "đã xử lý tới đâu". Ghi liên tục mỗi notice nên
-- tách khỏi sessions để không phải rewrite row đang chứa refresh_token.
-- history_id NULL = watch đã đăng ký nhưng chưa biết checkpoint (Gmail không trả
-- historyId, hoặc mới chỉ gia hạn watch); worker sẽ hỏi users.getProfile.
CREATE TABLE mail_provider.noti_indexes (
    id               uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id       uuid        NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    history_id       varchar(32),
    watch_expires_at timestamptz,
    last_notice_at   timestamptz,
    created_at       timestamptz NOT NULL DEFAULT now(),
    updated_at       timestamptz NOT NULL DEFAULT now(),
    -- historyId là uint64 opaque: chỉ nhận chữ số (khớp model.isDigits trong
    -- model/notice.go). Không dùng bigint vì uint64 có thể vượt max của bigint.
    CONSTRAINT ck_mail_provider_noti_indexes_history_id_digits
        CHECK (history_id IS NULL OR history_id ~ '^[0-9]+$'),
    -- 1 profile 1 checkpoint (giả định 1 hộp thư / profile). Nếu sau này watch
    -- nhiều hộp thư thì bỏ unique này và chuyển khoá sang (profile_id, email).
    CONSTRAINT uq_mail_provider_noti_indexes_profile
        UNIQUE (profile_id)
);

-- renewLoop quét các watch sắp hết hạn (<= MAIL_WATCH_RENEW_THRESHOLD).
CREATE INDEX IF NOT EXISTS ix_mail_provider_noti_indexes_watch_expires
    ON mail_provider.noti_indexes (watch_expires_at);

-- ------------------------------------------------------------------ updated_at
CREATE OR REPLACE FUNCTION mail_provider.fn_set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_mail_provider_sessions_updated_at ON mail_provider.sessions;
CREATE TRIGGER trg_mail_provider_sessions_updated_at
    BEFORE UPDATE ON mail_provider.sessions
    FOR EACH ROW EXECUTE FUNCTION mail_provider.fn_set_updated_at();

DROP TRIGGER IF EXISTS trg_mail_provider_noti_indexes_updated_at ON mail_provider.noti_indexes;
CREATE TRIGGER trg_mail_provider_noti_indexes_updated_at
    BEFORE UPDATE ON mail_provider.noti_indexes
    FOR EACH ROW EXECUTE FUNCTION mail_provider.fn_set_updated_at();
