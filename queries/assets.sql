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