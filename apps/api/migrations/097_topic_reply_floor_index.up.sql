-- 097: index topic_reply for floor-order reads.
--
-- The W2 census found no (topic_id, floor) index: listTopicReplies and the
-- from_floor anchor ORDER BY (topic_id, floor, id) and were scanning the table.
-- This index is additive (CREATE INDEX IF NOT EXISTS), so deploy order against
-- the new readers does not matter. A partial index on status = 0 is not needed:
-- production had three hidden rows.

CREATE INDEX IF NOT EXISTS idx_topic_reply_topic_floor ON topic_reply (topic_id, floor, id);
