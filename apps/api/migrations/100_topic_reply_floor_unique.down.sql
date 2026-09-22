-- Drops the unique (topic_id, floor) index and topic.last_reply_floor.
-- Floor renumbering is not undone: replies that were shifted keep their
-- new floors, so rolling this back does not restore the pre-098 duplicates.
DROP INDEX IF EXISTS uq_topic_reply_topic_floor;
ALTER TABLE topic DROP COLUMN IF EXISTS last_reply_floor;
ALTER TABLE topic_section_relation DROP COLUMN IF EXISTS position;
