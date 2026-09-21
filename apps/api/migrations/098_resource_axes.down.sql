DROP INDEX IF EXISTS idx_galgame_resource_runtimes;
DROP INDEX IF EXISTS idx_galgame_resource_platforms;

ALTER TABLE galgame_resource
  DROP COLUMN IF EXISTS runtimes,
  DROP COLUMN IF EXISTS platforms,
  DROP COLUMN IF EXISTS languages,
  DROP COLUMN IF EXISTS version_label,
  DROP COLUMN IF EXISTS title;
