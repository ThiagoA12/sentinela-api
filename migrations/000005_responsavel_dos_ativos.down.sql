DROP INDEX IF EXISTS assets_owner_idx;

ALTER TABLE assets
    DROP COLUMN IF EXISTS created_by,
    DROP COLUMN IF EXISTS owner_id;
