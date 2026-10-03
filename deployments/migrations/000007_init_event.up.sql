-- =============================================================================
-- 000007_init_event
-- event service: events (sự kiện cá nhân của từng profile)
-- Phụ thuộc: 000001_init_identity (identity.profiles)
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS event;

-- ==================================================================== events
-- Một row = một sự kiện. Khác `calendar.schedules` ở `status` + `source`:
-- sự kiện có vòng đời (planned/confirmed/cancelled) và truy vết được nguồn
-- (email nào sinh ra nó).
CREATE TABLE event.events (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid         NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    title       varchar(500) NOT NULL,
    description text,
    location    varchar(255),
    start_at    timestamptz  NOT NULL,
    end_at      timestamptz,
    all_day     boolean      NOT NULL DEFAULT false,
    color       varchar(9),
    -- Vòng đời sự kiện; DB chỉ chấp nhận 3 giá trị đã biết.
    status      varchar(20)  NOT NULL DEFAULT 'PLANNED',
    -- Id email nguồn (rỗng/NULL = người dùng tạo trên UI).
    source      varchar(255),
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_event_events_title_not_blank
        CHECK (btrim(title) <> ''),
    -- end_at NULL hợp lệ; nếu có thì không được sớm hơn start_at.
    CONSTRAINT ck_event_events_time_range
        CHECK (end_at IS NULL OR end_at >= start_at),
    CONSTRAINT ck_event_events_color_format
        CHECK (color IS NULL OR color ~* '^#[0-9a-f]{6}([0-9a-f]{2})?$'),
    CONSTRAINT ck_event_events_status
        CHECK (status IN ('PLANNED', 'CONFIRMED', 'CANCELLED'))
);

-- Truy vấn chính: sự kiện của 1 profile trong 1 khoảng thời gian, sắp theo start_at.
CREATE INDEX IF NOT EXISTS ix_event_events_profile_start
    ON event.events (profile_id, start_at);

-- ------------------------------------------------------------------ updated_at
CREATE OR REPLACE FUNCTION event.fn_set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_event_events_updated_at ON event.events;
CREATE TRIGGER trg_event_events_updated_at
    BEFORE UPDATE ON event.events
    FOR EACH ROW EXECUTE FUNCTION event.fn_set_updated_at();
