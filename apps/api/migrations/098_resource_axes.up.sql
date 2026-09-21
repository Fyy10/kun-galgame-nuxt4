-- 098: richer resource axes aligned with LetMoe's user-facing vocab.
--
-- title / version_label are optional free text. languages / platforms / runtimes
-- are jsonb arrays (LetMoe keys). The old varchar type/language/platform columns
-- stay: the galgame browse filters still equality-match them, and writes keep
-- a compatibility scalar next to the arrays.
--
-- Existing rows: language others → ["other"]; windows → platforms ["win"] and
-- runtimes ["native-win"]; app → ["and"] + ["native-and"]; mac/linux/others map
-- to catalog keys. emulator is left as [] — those 3459 rows hid the runtime in
-- the note and the size field, and a later cmd fills them. Re-running the
-- UPDATEs is gated on empty arrays so a second apply is a no-op.

ALTER TABLE galgame_resource
  ADD COLUMN IF NOT EXISTS title varchar(200) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS version_label varchar(64) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS languages jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS platforms jsonb NOT NULL DEFAULT '[]'::jsonb,
  ADD COLUMN IF NOT EXISTS runtimes jsonb NOT NULL DEFAULT '[]'::jsonb;

UPDATE galgame_resource
SET languages = CASE
  WHEN language = 'others' THEN '["other"]'::jsonb
  WHEN language IS NULL OR language = '' OR language = 'all' THEN '[]'::jsonb
  ELSE jsonb_build_array(language)
END
WHERE languages = '[]'::jsonb AND language IS NOT NULL AND language <> '';

UPDATE galgame_resource
SET
  platforms = CASE platform
    WHEN 'windows' THEN '["win"]'::jsonb
    WHEN 'mac' THEN '["mac"]'::jsonb
    WHEN 'linux' THEN '["lin"]'::jsonb
    WHEN 'app' THEN '["and"]'::jsonb
    WHEN 'others' THEN '["oth"]'::jsonb
    ELSE '[]'::jsonb
  END,
  runtimes = CASE platform
    WHEN 'windows' THEN '["native-win"]'::jsonb
    WHEN 'app' THEN '["native-and"]'::jsonb
    ELSE '[]'::jsonb
  END
WHERE platforms = '[]'::jsonb
  AND platform IN ('windows', 'mac', 'linux', 'app', 'others');

CREATE INDEX IF NOT EXISTS idx_galgame_resource_platforms
  ON galgame_resource USING gin (platforms jsonb_path_ops);
CREATE INDEX IF NOT EXISTS idx_galgame_resource_runtimes
  ON galgame_resource USING gin (runtimes jsonb_path_ops);
