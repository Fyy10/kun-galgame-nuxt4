-- 101: index topic_comment for the comment read face.
--
-- The face selects WHERE topic_reply_id IN (...) AND status = 0
-- ORDER BY created, id, and no index started with topic_reply_id: the table
-- had (created), (topic_id, created), a partial (parent_comment_id) and a
-- content trigram index. The index is additive, so deploy order does not
-- matter.
--
-- Two columns are deliberately left alone:
--
--   target_user_id: the 6 top-level rows whose value is not their reply's
--   author are not drift. In every one of them the commenter IS the reply's
--   author and the target is a third party they picked in the "comment at
--   someone" UI that has since been retired. Rewriting them would destroy
--   real data. The v1 write face derives the value instead; the read face
--   sends the stored column as it is.
--
--   topic_reply.comment_count: nothing has maintained it since 2026-06, and
--   it goes away in 102 (deploy-then-drop).

CREATE INDEX IF NOT EXISTS idx_topic_comment_reply_created_id
  ON topic_comment (topic_reply_id, created, id)
  WHERE status = 0;
