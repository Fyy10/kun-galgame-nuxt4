-- 096 down drops the CHECK only. The three rows that held 2 or 3 cannot be
-- told apart from the thousands of genuine 0s after the up migration, and
-- those values had no meaning in application code, so restoring them would
-- invent history.
ALTER TABLE topic DROP CONSTRAINT IF EXISTS topic_status_check;
