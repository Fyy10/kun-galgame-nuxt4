-- 099: recount topic and reply engagement counters; add history-list indexes.
--
-- Concurrent likes used to increment like_count without a RETURNING guard, so a
-- duplicate insert still added 1. Favorite and upvote counts drifted the same
-- way. Recount from the live row tables and write only rows whose cached value
-- differs. Pairs that hold both a like and a dislike are left as they are:
-- there is no way to know which was meant; the next set of one token removes
-- the other. The three indexes serve the v1 history lists (created DESC, id
-- DESC). This UPDATE does not set topic.updated / topic_reply.updated.

UPDATE topic t
SET like_count = s.n
FROM (
  SELECT t2.id, COALESCE(c.n, 0)::int AS n
  FROM topic t2
  LEFT JOIN (
    SELECT topic_id AS id, COUNT(*)::int AS n
    FROM topic_reaction
    WHERE reaction = 'like'
    GROUP BY topic_id
  ) c ON c.id = t2.id
) s
WHERE t.id = s.id AND t.like_count IS DISTINCT FROM s.n;

UPDATE topic t
SET dislike_count = s.n
FROM (
  SELECT t2.id, COALESCE(c.n, 0)::int AS n
  FROM topic t2
  LEFT JOIN (
    SELECT topic_id AS id, COUNT(*)::int AS n
    FROM topic_reaction
    WHERE reaction = 'dislike'
    GROUP BY topic_id
  ) c ON c.id = t2.id
) s
WHERE t.id = s.id AND t.dislike_count IS DISTINCT FROM s.n;

UPDATE topic t
SET favorite_count = s.n
FROM (
  SELECT t2.id, COALESCE(c.n, 0)::int AS n
  FROM topic t2
  LEFT JOIN (
    SELECT topic_id AS id, COUNT(*)::int AS n
    FROM topic_favorite
    GROUP BY topic_id
  ) c ON c.id = t2.id
) s
WHERE t.id = s.id AND t.favorite_count IS DISTINCT FROM s.n;

UPDATE topic t
SET upvote_count = s.n
FROM (
  SELECT t2.id, COALESCE(c.n, 0)::int AS n
  FROM topic t2
  LEFT JOIN (
    SELECT topic_id AS id, COUNT(*)::int AS n
    FROM topic_upvote
    GROUP BY topic_id
  ) c ON c.id = t2.id
) s
WHERE t.id = s.id AND t.upvote_count IS DISTINCT FROM s.n;

UPDATE topic_reply r
SET like_count = s.n
FROM (
  SELECT r2.id, COALESCE(c.n, 0)::int AS n
  FROM topic_reply r2
  LEFT JOIN (
    SELECT topic_reply_id AS id, COUNT(*)::int AS n
    FROM topic_reply_reaction
    WHERE reaction = 'like'
    GROUP BY topic_reply_id
  ) c ON c.id = r2.id
) s
WHERE r.id = s.id AND r.like_count IS DISTINCT FROM s.n;

UPDATE topic_reply r
SET dislike_count = s.n
FROM (
  SELECT r2.id, COALESCE(c.n, 0)::int AS n
  FROM topic_reply r2
  LEFT JOIN (
    SELECT topic_reply_id AS id, COUNT(*)::int AS n
    FROM topic_reply_reaction
    WHERE reaction = 'dislike'
    GROUP BY topic_reply_id
  ) c ON c.id = r2.id
) s
WHERE r.id = s.id AND r.dislike_count IS DISTINCT FROM s.n;

CREATE INDEX IF NOT EXISTS idx_topic_upvote_topic_created_id
  ON topic_upvote (topic_id, created DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_topic_reaction_topic_created_id
  ON topic_reaction (topic_id, created DESC, id DESC);
CREATE INDEX IF NOT EXISTS idx_topic_reply_reaction_reply_created_id
  ON topic_reply_reaction (topic_reply_id, created DESC, id DESC);
