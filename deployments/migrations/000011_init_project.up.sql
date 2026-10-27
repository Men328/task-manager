-- =============================================================================
-- 000011_init_project
-- project service: projects (nhóm công việc của từng profile) + cột project_id
-- bắt buộc trên task.TASKS, calendar.schedules, event.events.
-- Phụ thuộc: 000001_init_identity, 000002_init_task, 000006_init_calendar,
--            000007_init_event
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS project;

-- =================================================================== projects
-- Một row = một project. Mỗi profile có đúng 1 project mặc định ("Chung").
-- `description` dùng làm ngữ cảnh để auto-detect project cho item mới.
CREATE TABLE project.projects (
    id          uuid         PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid         NOT NULL REFERENCES identity.profiles (id) ON DELETE CASCADE,
    name        varchar(100) NOT NULL,
    description text         NOT NULL DEFAULT '',
    color       varchar(9),
    is_default  boolean      NOT NULL DEFAULT false,
    is_archived boolean      NOT NULL DEFAULT false,
    created_at  timestamptz  NOT NULL DEFAULT now(),
    updated_at  timestamptz  NOT NULL DEFAULT now(),
    CONSTRAINT ck_project_projects_name_not_blank
        CHECK (btrim(name) <> ''),
    CONSTRAINT ck_project_projects_color_format
        CHECK (color IS NULL OR color ~* '^#[0-9a-f]{6}([0-9a-f]{2})?$'),
    -- Cho phép FK composite (id, profile_id) từ các bảng item:
    -- đảm bảo item và project luôn cùng profile.
    CONSTRAINT uq_projects_id_profile UNIQUE (id, profile_id)
);

-- Mỗi profile đúng 1 project mặc định.
CREATE UNIQUE INDEX IF NOT EXISTS uq_projects_one_default
    ON project.projects (profile_id) WHERE is_default;

-- Tên không trùng trong profile (không phân biệt hoa thường).
CREATE UNIQUE INDEX IF NOT EXISTS uq_projects_profile_name
    ON project.projects (profile_id, lower(name));

CREATE INDEX IF NOT EXISTS ix_project_projects_profile_archived
    ON project.projects (profile_id, is_archived);

-- ------------------------------------------------------------------ updated_at
CREATE OR REPLACE FUNCTION project.fn_set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_project_projects_updated_at ON project.projects;
CREATE TRIGGER trg_project_projects_updated_at
    BEFORE UPDATE ON project.projects
    FOR EACH ROW EXECUTE FUNCTION project.fn_set_updated_at();

-- ------------------------------------------------------------- backfill (1)
-- Tạo project mặc định "Chung" cho mọi profile hiện có.
INSERT INTO project.projects (profile_id, name, is_default)
SELECT id, 'Chung', true FROM identity.profiles;

-- ------------------------------------------------- project_id: thêm cột (2)
ALTER TABLE task.TASKS          ADD COLUMN project_id uuid;
ALTER TABLE calendar.schedules  ADD COLUMN project_id uuid;
ALTER TABLE event.events        ADD COLUMN project_id uuid;

-- ------------------------------------------------- gán vào project "Chung" (3)
UPDATE task.TASKS t SET project_id = p.id
  FROM project.projects p WHERE p.profile_id = t.profile_id AND p.is_default;
UPDATE calendar.schedules s SET project_id = p.id
  FROM project.projects p WHERE p.profile_id = s.profile_id AND p.is_default;
UPDATE event.events e SET project_id = p.id
  FROM project.projects p WHERE p.profile_id = e.profile_id AND p.is_default;

-- ------------------------------------------------- NOT NULL + FK composite (4)
ALTER TABLE task.TASKS          ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE calendar.schedules  ALTER COLUMN project_id SET NOT NULL;
ALTER TABLE event.events        ALTER COLUMN project_id SET NOT NULL;

ALTER TABLE task.TASKS
    ADD CONSTRAINT fk_tasks_project FOREIGN KEY (project_id, profile_id)
        REFERENCES project.projects (id, profile_id) ON DELETE RESTRICT;
ALTER TABLE calendar.schedules
    ADD CONSTRAINT fk_calendar_schedules_project FOREIGN KEY (project_id, profile_id)
        REFERENCES project.projects (id, profile_id) ON DELETE RESTRICT;
ALTER TABLE event.events
    ADD CONSTRAINT fk_event_events_project FOREIGN KEY (project_id, profile_id)
        REFERENCES project.projects (id, profile_id) ON DELETE RESTRICT;

-- ------------------------------------------------------------------ index (5)
CREATE INDEX IF NOT EXISTS ix_tasks_profile_project
    ON task.TASKS (profile_id, project_id);
CREATE INDEX IF NOT EXISTS ix_calendar_schedules_profile_project
    ON calendar.schedules (profile_id, project_id);
CREATE INDEX IF NOT EXISTS ix_event_events_profile_project
    ON event.events (profile_id, project_id);
