-- 095: mirror community-service notification rows onto message.
--
-- Adds the columns the feed poller upserts by (community_notification_id, seq,
-- thread, post number, fold counts). Existing rows keep NULL / the defaults of 1
-- so the local inbox is unchanged until the poller writes a mirrored row.

ALTER TABLE message ADD COLUMN IF NOT EXISTS community_notification_id bigint NULL;
ALTER TABLE message ADD COLUMN IF NOT EXISTS community_seq bigint NULL;
ALTER TABLE message ADD COLUMN IF NOT EXISTS community_thread_id bigint NULL;
ALTER TABLE message ADD COLUMN IF NOT EXISTS community_post_number integer NULL;
ALTER TABLE message ADD COLUMN IF NOT EXISTS item_count integer NOT NULL DEFAULT 1;
ALTER TABLE message ADD COLUMN IF NOT EXISTS actor_count integer NOT NULL DEFAULT 1;

CREATE UNIQUE INDEX IF NOT EXISTS message_community_notification_id_key
  ON message (community_notification_id)
  WHERE community_notification_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS message_receiver_community_thread_idx
  ON message (receiver_id, community_thread_id)
  WHERE community_thread_id IS NOT NULL;
