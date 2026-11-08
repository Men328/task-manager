-- =============================================================================
-- 000017_init_message
-- message: tin nhắn 1-1 giữa bạn bè.
-- message-service chỉ ĐỌC các bảng này; worker-service (Kafka consumer topic
-- chat.messages) là nơi DUY NHẤT ghi. Ghi idempotent theo id để chịu được
-- at-least-once của Kafka.
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS message;

-- id của cuộc trò chuyện suy ra tất định từ cặp profile (UUIDv5), member_a < member_b.
CREATE TABLE IF NOT EXISTS message.conversations (
    id              uuid        PRIMARY KEY,
    member_a        uuid        NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    member_b        uuid        NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    last_message_id uuid        NOT NULL,
    last_message_at timestamptz NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_conversations_order CHECK (member_a < member_b)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_conversations_pair ON message.conversations (member_a, member_b);
CREATE INDEX IF NOT EXISTS ix_conversations_member_a ON message.conversations (member_a, last_message_at DESC);
CREATE INDEX IF NOT EXISTS ix_conversations_member_b ON message.conversations (member_b, last_message_at DESC);

CREATE TABLE IF NOT EXISTS message.messages (
    id              uuid         PRIMARY KEY,
    conversation_id uuid         NOT NULL REFERENCES message.conversations (id) ON DELETE CASCADE,
    sender_id       uuid         NOT NULL,
    recipient_id    uuid         NOT NULL,
    body            text         NOT NULL,
    client_id       varchar(64)  NOT NULL DEFAULT '',
    created_at      timestamptz  NOT NULL,
    persisted_at    timestamptz  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS ix_messages_conversation ON message.messages (conversation_id, created_at DESC, id DESC);
