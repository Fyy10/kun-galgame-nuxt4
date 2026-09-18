-- 096: collapse leftover topic.status values 2 and 3 into 0 (published), then
-- allow only 0 and 1.
--
-- Production on 2026-09-18 had 3,219 rows at 0, 313 at 1 (hidden), and three
-- 2024 leftovers at 2 or 3 (ids 1712, 1824, 1160). No current code assigns
-- meaning to 2 or 3; read_decision.go only tests status == 1, and hidden_by is
-- '' on all three. v1 maps 0 → published and 1 → hidden and treats any other
-- value as a data defect (500), so these rows would 500 the list until they
-- are 0. Existing 0 and 1 rows are unchanged.
--
-- The CHECK keeps the defect from coming back: hideDecision is the only writer
-- and it writes 0 or 1, so old and new code both satisfy it in either deploy
-- order.

UPDATE topic SET status = 0 WHERE status IN (2, 3);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'topic_status_check') THEN
    ALTER TABLE topic ADD CONSTRAINT topic_status_check CHECK (status IN (0, 1));
  END IF;
END $$;
