DROP INDEX IF EXISTS assets_identifier_active_key;

ALTER TABLE assets
DROP COLUMN IF EXISTS version,
DROP COLUMN IF EXISTS deleted_at,
DROP COLUMN IF EXISTS updated_at,
DROP COLUMN IF EXISTS description;