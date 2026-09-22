-- The column comes back empty: its content was a 2026-06 snapshot that had
-- already rotted, and no code maintains it.
ALTER TABLE topic_reply ADD COLUMN IF NOT EXISTS comment_count INT NOT NULL DEFAULT 0;
