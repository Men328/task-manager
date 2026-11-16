DELETE FROM notification.notices WHERE target_type = 'message';
ALTER TABLE notification.notices DROP CONSTRAINT IF EXISTS ck_notification_notices_target_type;
ALTER TABLE notification.notices ADD CONSTRAINT ck_notification_notices_target_type
    CHECK (target_type IN ('task', 'schedule', 'event', 'backlog', 'friend'));

ALTER TABLE message.messages
    DROP COLUMN IF EXISTS task_status_changed_at,
    DROP COLUMN IF EXISTS task_status_color,
    DROP COLUMN IF EXISTS task_status_category,
    DROP COLUMN IF EXISTS task_status_name,
    DROP COLUMN IF EXISTS task_title,
    DROP COLUMN IF EXISTS task_id;

ALTER TABLE task.tasks
    DROP COLUMN IF EXISTS source_message_id,
    DROP COLUMN IF EXISTS requested_by;
