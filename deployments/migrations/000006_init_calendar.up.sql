-- =============================================================================
-- 000006_init_calendar
-- calendar service: schedules (lịch cá nhân của từng profile)
-- Phụ thuộc: 000001_init_identity (identity.profiles)
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS calendar;

-- ================================================================= schedules
-- Một row = một lịch (event). Thuộc sở hữu của profile_id.
-- start_at bắt buộc; end_at NULL = mốc thời gian không có thời lượng.
-- all_day = true thì UI hiển thị cả ngày (FullCalendar allDay).
CREATE TABLE calendar.schedules (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid         NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    title       varchar(500) NOT NULL,
    description text,
    location    varchar(255),
    start_at    timestamptz  NOT NULL,
    end_at      timestamptz,
    all_day     boolean      NOT NULL DEFAULT false,
    color       varchar(9),
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_calendar_schedules_title_not_blank
        CHECK (btrim(title) <> ''),
    -- end_at NULL hợp lệ; nếu có thì không được sớm hơn start_at.
    CONSTRAINT ck_calendar_schedules_time_range
        CHECK (end_at IS NULL OR end_at >= start_at),
    CONSTRAINT ck_calendar_schedules_color_format
        CHECK (color IS NULL OR color ~* '^#[0-9a-f]{6}([0-9a-f]{2})?$')
);

-- Truy vấn chính: lịch của 1 profile trong 1 khoảng thời gian, sắp theo start_at.
CREATE INDEX IF NOT EXISTS ix_calendar_schedules_profile_start
    ON calendar.schedules (profile_id, start_at);

-- ------------------------------------------------------------------ updated_at
CREATE OR REPLACE FUNCTION calendar.fn_set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_calendar_schedules_updated_at ON calendar.schedules;
CREATE TRIGGER trg_calendar_schedules_updated_at
    BEFORE UPDATE ON calendar.schedules
    FOR EACH ROW EXECUTE FUNCTION calendar.fn_set_updated_at();
