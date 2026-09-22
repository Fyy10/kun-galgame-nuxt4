-- 110: give the topic-draft keyset an id tie-breaker.
--
-- idx_topic_draft_user was (user_id, updated DESC). The v1 list face is a
-- cursor collection sorted updated DESC, id DESC, and without the id column
-- the index stops one key short: the planner can still use it, but two drafts
-- saved in the same second have no stable order, so a page boundary between
-- them drops or repeats a row.
--
-- Production has no ties on (user_id, updated) today, which is why nothing has
-- gone wrong yet and why a test seeded from production-shaped data would not
-- catch the missing tie-breaker. W4 lost three pagination tests to exactly
-- this: every seeded row had a distinct second, so deleting the id key from
-- the keyset left all three green.
--
-- The old index is a strict prefix of the new one, so nothing that used it
-- loses its access path. Additive then drop, in one transaction.

CREATE INDEX IF NOT EXISTS idx_topic_draft_user_updated_id
  ON topic_draft (user_id, updated DESC, id DESC);

DROP INDEX IF EXISTS idx_topic_draft_user;
