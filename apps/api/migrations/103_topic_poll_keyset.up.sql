-- 103: index the poll tables for the v1 poll faces.
--
-- GET /api/v1/polls/{poll_id}/votes walks topic_poll_vote by keyset on
-- (created, id) within one poll, and the only index that starts with poll_id
-- is the unique (poll_id, option_id, user_id), which cannot answer that
-- ordering. GET /api/v1/topics/{topic_id}/polls orders topic_poll by
-- (created DESC, id DESC) within one topic, and topic_poll had no index on
-- topic_id at all.
--
-- Both are additive, so deploy order does not matter, and nothing here
-- touches a row: production holds 35 polls, 0 rows of cross-poll votes and 0
-- rows of vote_count drift (census §5.9), so there is no data to repair.

CREATE INDEX IF NOT EXISTS idx_topic_poll_vote_poll_created_id
  ON topic_poll_vote (poll_id, created, id);

CREATE INDEX IF NOT EXISTS idx_topic_poll_topic_created_id
  ON topic_poll (topic_id, created DESC, id DESC);
