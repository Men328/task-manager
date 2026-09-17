-- =============================================================================
-- 000002_init_task
-- task service: TASK_STATUSES, STATUS_TRANSITIONS, TASKS, TASK_STATUS_LOGS
-- Phụ thuộc: 000001_init_identity (schema identity)
-- =============================================================================

CREATE SCHEMA IF NOT EXISTS task;

-- ------------------------------------------------------------------- enums
CREATE TYPE task.task_priority   AS ENUM ('low', 'medium', 'high', 'urgent');
CREATE TYPE task.status_category AS ENUM ('todo', 'in_progress', 'done', 'cancelled');

-- =========================================================== TASK_STATUSES
-- Status do từng profile tự định nghĩa (1-n).
CREATE TABLE task.TASK_STATUSES (
    id          uuid                  PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id  uuid                  NOT NULL REFERENCES identity.PROFILES (id) ON DELETE CASCADE,
    name        varchar(50)           NOT NULL,
    slug        varchar(50)           NOT NULL,
    description text,
    color       varchar(9),
    category    task.status_category  NOT NULL DEFAULT 'todo',
    is_default  boolean               NOT NULL DEFAULT false,
    is_terminal boolean               NOT NULL DEFAULT false,
    position    integer               NOT NULL DEFAULT 0,
    is_archived boolean               NOT NULL DEFAULT false,
    created_at  timestamptz           NOT NULL DEFAULT now(),
    updated_at  timestamptz           NOT NULL DEFAULT now(),
    CONSTRAINT ck_task_statuses_name_not_blank CHECK (btrim(name) <> ''),
    CONSTRAINT ck_task_statuses_slug_format    CHECK (slug ~ '^[a-z0-9][a-z0-9_-]*$'),
    CONSTRAINT ck_task_statuses_color_format   CHECK (color IS NULL OR color ~* '^#[0-9a-f]{6}([0-9a-f]{2})?$'),
    CONSTRAINT uq_task_statuses_id_profile     UNIQUE (id, profile_id),
    CONSTRAINT uq_task_statuses_profile_slug   UNIQUE (profile_id, slug)
);

-- Mỗi profile chỉ có 1 status default đang hoạt động (DBML không diễn đạt được partial index)
CREATE UNIQUE INDEX IF NOT EXISTS uq_task_statuses_one_default
    ON task.TASK_STATUSES (profile_id)
    WHERE is_default AND NOT is_archived;

CREATE INDEX IF NOT EXISTS ix_task_statuses_profile_position
    ON task.TASK_STATUSES (profile_id, position);

-- ======================================================= STATUS_TRANSITIONS
-- Lifecycle = ALLOWLIST: có row => được phép chuyển from -> to, không có => cấm.
CREATE TABLE task.STATUS_TRANSITIONS (
    id             uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id     uuid        NOT NULL REFERENCES identity.PROFILES (id) ON DELETE CASCADE,
    from_status_id uuid        NOT NULL,
    to_status_id   uuid        NOT NULL,
    is_active      boolean     NOT NULL DEFAULT true,
    requires_note  boolean     NOT NULL DEFAULT false,
    guard          jsonb,
    description    text,
    created_at     timestamptz NOT NULL DEFAULT now(),
    updated_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT ck_status_transitions_not_self CHECK (from_status_id <> to_status_id),
    CONSTRAINT uq_status_transitions_from_to   UNIQUE (from_status_id, to_status_id),
    -- composite FK: đảm bảo 2 status thuộc cùng profile
    CONSTRAINT fk_status_transitions_from FOREIGN KEY (from_status_id, profile_id)
        REFERENCES task.TASK_STATUSES (id, profile_id) ON DELETE CASCADE,
    CONSTRAINT fk_status_transitions_to FOREIGN KEY (to_status_id, profile_id)
        REFERENCES task.TASK_STATUSES (id, profile_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS ix_status_transitions_profile_from
    ON task.STATUS_TRANSITIONS (profile_id, from_status_id);

-- ==================================================================== TASKS
CREATE TABLE task.TASKS (
    id             uuid               PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id     uuid               NOT NULL REFERENCES identity.PROFILES (id) ON DELETE CASCADE,
    parent_task_id uuid,
    status_id      uuid               NOT NULL,
    title          varchar(500)       NOT NULL,
    description    text,
    priority       task.task_priority NOT NULL DEFAULT 'medium',
    position       integer            NOT NULL DEFAULT 0,
    start_at       timestamptz,
    due_at         timestamptz,
    completed_at   timestamptz,
    is_archived    boolean            NOT NULL DEFAULT false,
    created_at     timestamptz        NOT NULL DEFAULT now(),
    updated_at     timestamptz        NOT NULL DEFAULT now(),
    deleted_at     timestamptz,
    CONSTRAINT ck_tasks_title_not_blank  CHECK (btrim(title) <> ''),
    CONSTRAINT ck_tasks_due_after_start  CHECK (start_at IS NULL OR due_at IS NULL OR due_at >= start_at),
    CONSTRAINT uq_tasks_id_profile       UNIQUE (id, profile_id),
    -- status phải thuộc cùng profile (tương ứng ON DELETE RESTRICT: không xoá status đang dùng)
    CONSTRAINT fk_tasks_status FOREIGN KEY (status_id, profile_id)
        REFERENCES task.TASK_STATUSES (id, profile_id) ON DELETE RESTRICT,
    -- task cha phải thuộc cùng profile; không cascade để tránh mất task con
    CONSTRAINT fk_tasks_parent FOREIGN KEY (parent_task_id, profile_id)
        REFERENCES task.TASKS (id, profile_id) ON DELETE NO ACTION
);

CREATE INDEX IF NOT EXISTS ix_tasks_profile_status   ON task.TASKS (profile_id, status_id);
CREATE INDEX IF NOT EXISTS ix_tasks_parent_position  ON task.TASKS (parent_task_id, position);
CREATE INDEX IF NOT EXISTS ix_tasks_profile_due      ON task.TASKS (profile_id, due_at);
CREATE INDEX IF NOT EXISTS ix_tasks_profile_active   ON task.TASKS (profile_id, is_archived, deleted_at);

-- ========================================================= TASK_STATUS_LOGS
CREATE TABLE task.TASK_STATUS_LOGS (
    id             bigserial   PRIMARY KEY,
    task_id        uuid        NOT NULL,
    profile_id     uuid        NOT NULL,
    from_status_id uuid,
    to_status_id   uuid        NOT NULL,
    note           text,
    changed_at     timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT fk_status_logs_task FOREIGN KEY (task_id, profile_id)
        REFERENCES task.TASKS (id, profile_id) ON DELETE CASCADE,
    CONSTRAINT fk_status_logs_from FOREIGN KEY (from_status_id, profile_id)
        REFERENCES task.TASK_STATUSES (id, profile_id) ON DELETE NO ACTION,
    CONSTRAINT fk_status_logs_to FOREIGN KEY (to_status_id, profile_id)
        REFERENCES task.TASK_STATUSES (id, profile_id) ON DELETE NO ACTION
);

CREATE INDEX IF NOT EXISTS ix_status_logs_task_time  ON task.TASK_STATUS_LOGS (task_id, changed_at);
CREATE INDEX IF NOT EXISTS ix_status_logs_profile_to ON task.TASK_STATUS_LOGS (profile_id, to_status_id);

-- ============================================ trigger: chặn vòng lặp cây task
-- Composite FK không chặn được A là con của B và B là con của A.
CREATE OR REPLACE FUNCTION task.fn_tasks_prevent_cycle() RETURNS trigger AS $$
DECLARE
    cur   uuid := NEW.parent_task_id;
    depth integer := 0;
BEGIN
    IF NEW.parent_task_id IS NULL THEN
        RETURN NEW;
    END IF;

    WHILE cur IS NOT NULL LOOP
        IF cur = NEW.id THEN
            RAISE EXCEPTION 'task cycle detected: task % không thể là con/cháu của chính nó', NEW.id
                USING ERRCODE = 'check_violation';
        END IF;

        depth := depth + 1;
        IF depth > 100 THEN
            RAISE EXCEPTION 'task tree quá sâu (>100) khi kiểm tra cycle' USING ERRCODE = 'check_violation';
        END IF;

        SELECT t.parent_task_id INTO cur FROM task.TASKS t WHERE t.id = cur;
    END LOOP;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_tasks_prevent_cycle ON task.TASKS;
CREATE TRIGGER trg_tasks_prevent_cycle
    BEFORE INSERT OR UPDATE ON task.TASKS
    FOR EACH ROW
    WHEN (NEW.parent_task_id IS NOT NULL)
    EXECUTE FUNCTION task.fn_tasks_prevent_cycle();

-- ============================================================ trigger updated_at
CREATE OR REPLACE FUNCTION task.fn_set_updated_at() RETURNS trigger AS $$
BEGIN
    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_task_statuses_updated_at ON task.TASK_STATUSES;
CREATE TRIGGER trg_task_statuses_updated_at
    BEFORE UPDATE ON task.TASK_STATUSES
    FOR EACH ROW EXECUTE FUNCTION task.fn_set_updated_at();

DROP TRIGGER IF EXISTS trg_status_transitions_updated_at ON task.STATUS_TRANSITIONS;
CREATE TRIGGER trg_status_transitions_updated_at
    BEFORE UPDATE ON task.STATUS_TRANSITIONS
    FOR EACH ROW EXECUTE FUNCTION task.fn_set_updated_at();

DROP TRIGGER IF EXISTS trg_tasks_updated_at ON task.TASKS;
CREATE TRIGGER trg_tasks_updated_at
    BEFORE UPDATE ON task.TASKS
    FOR EACH ROW EXECUTE FUNCTION task.fn_set_updated_at();
