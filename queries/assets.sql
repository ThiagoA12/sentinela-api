-- name: ListAssets :many
SELECT id, name, identifier, type, criticality, created_at,
    description, updated_at, deleted_at, version
FROM assets
WHERE deleted_at IS NULL
ORDER BY id
LIMIT 100;

-- name: CreateAsset :one
INSERT INTO assets (name, identifier, type, criticality, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, identifier, type, criticality, created_at,
    description, updated_at, deleted_at, version;

-- name: GetAsset :one
SELECT id, name, identifier, type, criticality, created_at,
    description, updated_at, deleted_at, version
FROM assets
WHERE id = $1 AND deleted_at IS NULL;

-- name: UpdateAsset :one
UPDATE assets
SET name = $3,
    identifier = $4,
    type = $5,
    criticality = $6,
    description = $7,
    version = version + 1,
    updated_at = now()
WHERE id = $1 AND version = $2 AND deleted_at IS NULL
RETURNING id, name, identifier, type, criticality, created_at,
    description, updated_at, deleted_at, version;

-- name: DeleteAsset :execrows
UPDATE assets
SET deleted_at = now(),
    updated_at = now(),
    version = version + 1
WHERE id = $1 AND deleted_at IS NULL;