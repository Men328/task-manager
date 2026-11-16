-- =============================================================================
-- 000018_chat_task
-- Giao task qua tin nhắn chat (lệnh /task):
--   - task.tasks nhớ người giao + tin nhắn nguồn để báo lại khi đổi trạng thái
--   - message.messages lưu ảnh chụp task (tiêu đề + trạng thái) để hiển thị thẻ
--   - notification cho phép notice trỏ tới tin nhắn
-- =============================================================================

ALTER TABLE task.tasks
    ADD COLUMN IF NOT EXISTS requested_by      uuid REFERENCES identity.profiles (id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS source_message_id uuid;

ALTER TABLE message.messages
    ADD COLUMN IF NOT EXISTS task_id                uuid,
    ADD COLUMN IF NOT EXISTS task_title             text,
    ADD COLUMN IF NOT EXISTS task_status_name       text,
    ADD COLUMN IF NOT EXISTS task_status_category   varchar(16),
    ADD COLUMN IF NOT EXISTS task_status_color      varchar(16),
    ADD COLUMN IF NOT EXISTS task_status_changed_at timestamptz;

ALTER TABLE notification.notices DROP CONSTRAINT IF EXISTS ck_notification_notices_target_type;
ALTER TABLE notification.notices ADD CONSTRAINT ck_notification_notices_target_type
    CHECK (target_type IN ('task', 'schedule', 'event', 'backlog', 'friend', 'message'));
