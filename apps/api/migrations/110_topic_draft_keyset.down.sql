CREATE INDEX IF NOT EXISTS idx_topic_draft_user
  ON topic_draft (user_id, updated DESC);

DROP INDEX IF EXISTS idx_topic_draft_user_updated_id;
