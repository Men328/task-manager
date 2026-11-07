-- =============================================================================
-- 000016_notice_friend_target
-- notification: cho phép notice trỏ tới lời mời kết bạn
-- =============================================================================

ALTER TABLE notification.notices DROP CONSTRAINT IF EXISTS ck_notification_notices_target_type;
ALTER TABLE notification.notices ADD CONSTRAINT ck_notification_notices_target_type
    CHECK (target_type IN ('task', 'schedule', 'event', 'backlog', 'friend'));
