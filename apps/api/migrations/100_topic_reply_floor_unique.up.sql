-- 100: unique reply floors, a per-topic floor counter, and an order for a
-- topic's sections.
--
-- CreateReply used MAX(floor)+1 inside a transaction but without a row lock
-- or a unique constraint, so two concurrent submits could share a floor.
-- Production had one clash: topic 122 floor 4, ids 202 and 203 (a double
-- submit 9ms apart). After this they become 4 and 5; the next two replies
-- on that topic shift from 5/6 to 6/7. Floors are never reused once assigned.
--
-- For every topic that has two replies on one floor, walk its replies in
-- (floor, id) order and set each floor to max(its floor, previous new floor
-- + 1). Topics without a duplicate are not touched, so existing gaps stay.
-- last_reply_floor is backfilled to MAX(floor) (0 when the topic has no
-- replies). reply_count is recounted from status = 0 rows only where it
-- differs. The unique index is created last so the renumbering can write.

DO $$
DECLARE
  tid integer;
  rec record;
  prev_floor integer;
  new_floor integer;
BEGIN
  FOR tid IN
    SELECT DISTINCT topic_id
    FROM topic_reply
    GROUP BY topic_id, floor
    HAVING COUNT(*) > 1
  LOOP
    prev_floor := 0;
    FOR rec IN
      SELECT id, floor
      FROM topic_reply
      WHERE topic_id = tid
      ORDER BY floor, id
    LOOP
      new_floor := GREATEST(rec.floor, prev_floor + 1);
      IF new_floor IS DISTINCT FROM rec.floor THEN
        UPDATE topic_reply SET floor = new_floor WHERE id = rec.id;
      END IF;
      prev_floor := new_floor;
    END LOOP;
  END LOOP;
END $$;

ALTER TABLE topic ADD COLUMN IF NOT EXISTS last_reply_floor integer NOT NULL DEFAULT 0;

UPDATE topic t
SET last_reply_floor = GREATEST(t.last_reply_floor, COALESCE((
  SELECT MAX(r.floor) FROM topic_reply r WHERE r.topic_id = t.id
), 0));

CREATE UNIQUE INDEX IF NOT EXISTS uq_topic_reply_topic_floor ON topic_reply (topic_id, floor);

-- topic_section_relation had no order column, and neither read of it had an
-- ORDER BY, so a topic's sections came back in whatever order Postgres chose
-- while the API documents them "in stored order". Existing rows are numbered
-- by section id, which is at least stable; new writes store the author's order.
ALTER TABLE topic_section_relation ADD COLUMN IF NOT EXISTS position smallint NOT NULL DEFAULT 0;

UPDATE topic_section_relation r
SET position = s.pos
FROM (
  SELECT topic_id, topic_section_id,
         (ROW_NUMBER() OVER (PARTITION BY topic_id ORDER BY topic_section_id) - 1)::smallint AS pos
  FROM topic_section_relation
) s
WHERE r.topic_id = s.topic_id AND r.topic_section_id = s.topic_section_id AND r.position IS DISTINCT FROM s.pos;

UPDATE topic t
SET reply_count = s.n
FROM (
  SELECT topic_id, COUNT(*)::integer AS n
  FROM topic_reply
  WHERE status = 0
  GROUP BY topic_id
) s
WHERE t.id = s.topic_id AND t.reply_count IS DISTINCT FROM s.n;

UPDATE topic t
SET reply_count = 0
WHERE t.reply_count IS DISTINCT FROM 0
  AND NOT EXISTS (
    SELECT 1 FROM topic_reply r WHERE r.topic_id = t.id AND r.status = 0
  );
