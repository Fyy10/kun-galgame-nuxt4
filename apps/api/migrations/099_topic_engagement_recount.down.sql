-- The recount of like / dislike / favorite / upvote counters is not undone:
-- the corrected values remain the source of truth. Only the history-list
-- indexes are dropped.

DROP INDEX IF EXISTS idx_topic_upvote_topic_created_id;
DROP INDEX IF EXISTS idx_topic_reaction_topic_created_id;
DROP INDEX IF EXISTS idx_topic_reply_reaction_reply_created_id;
