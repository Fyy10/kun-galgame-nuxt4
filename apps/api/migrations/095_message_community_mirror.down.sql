DROP INDEX IF EXISTS message_receiver_community_thread_idx;
DROP INDEX IF EXISTS message_community_notification_id_key;

ALTER TABLE message DROP COLUMN IF EXISTS actor_count;
ALTER TABLE message DROP COLUMN IF EXISTS item_count;
ALTER TABLE message DROP COLUMN IF EXISTS community_post_number;
ALTER TABLE message DROP COLUMN IF EXISTS community_thread_id;
ALTER TABLE message DROP COLUMN IF EXISTS community_seq;
ALTER TABLE message DROP COLUMN IF EXISTS community_notification_id;
