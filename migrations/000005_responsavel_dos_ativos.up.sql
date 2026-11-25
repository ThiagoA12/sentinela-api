ALTER TABLE assets
    ADD COLUMN owner_id   uuid REFERENCES users (id),
    ADD COLUMN created_by uuid REFERENCES users (id);

UPDATE assets
SET owner_id = adm.id,
    created_by = adm.id
FROM (
    SELECT id FROM users WHERE role = 'admin' ORDER BY created_at, id LIMIT 1
) AS adm;

ALTER TABLE assets
    ALTER COLUMN owner_id SET NOT NULL,
    ALTER COLUMN created_by SET NOT NULL;

CREATE INDEX assets_owner_idx ON assets (owner_id);
