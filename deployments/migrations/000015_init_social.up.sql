-- =============================================================================
-- 000015_init_social
-- social service: quan hệ bạn bè giữa các profile
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS social;

CREATE TABLE IF NOT EXISTS social.friendships (
    id           uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    requester_id uuid        NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    addressee_id uuid        NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    status       varchar(16) NOT NULL DEFAULT 'pending',
    created_at   timestamptz NOT NULL DEFAULT now(),
    accepted_at  timestamptz,
    CONSTRAINT ck_friendships_not_self CHECK (requester_id <> addressee_id),
    CONSTRAINT ck_friendships_status   CHECK (status IN ('pending', 'accepted'))
);

-- Mỗi cặp chỉ có 1 dòng, bất kể ai là người gửi.
CREATE UNIQUE INDEX IF NOT EXISTS uq_friendships_pair ON social.friendships
    (LEAST(requester_id, addressee_id), GREATEST(requester_id, addressee_id));

CREATE INDEX IF NOT EXISTS ix_friendships_requester ON social.friendships (requester_id, status);
CREATE INDEX IF NOT EXISTS ix_friendships_addressee ON social.friendships (addressee_id, status);
