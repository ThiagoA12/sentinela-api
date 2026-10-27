ALTER TABLE assets
    ADD COLUMN description varchar(2000),
    ADD COLUMN updated_at   timestamptz NOT NULL DEFAULT now(),
    ADD COLUMN deleted_at  timestamptz,
    ADD COLUMN version     integer     NOT NULL DEFAULT 1;

UPDATE assets SET updated_at = created_at;

CREATE UNIQUE INDEX assets_identifier_active_key
    ON assets (identifier)
    WHERE deleted_at IS NULL;