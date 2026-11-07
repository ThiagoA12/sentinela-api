-- name: ListAssets :many
SELECT id, name, identifier, type, criticality, created_at,
    description, updated_at, deleted_at, version
FROM assets
WHERE deleted_at IS NULL
    AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
    AND (sqlc.narg('criticality')::text IS NULL OR criticality = sqlc.narg('criticality'))
    AND (sqlc.narg('q')::text IS NULL
        OR name ILIKE '%' || sqlc.narg('q') || '%' ESCAPE '\'
        OR identifier ILIKE '%' || sqlc.narg('q') || '%' ESCAPE '\')
ORDER BY
    CASE WHEN sqlc.arg('sort')::text = 'name' THEN name END,
    CASE WHEN sqlc.arg('sort')::text = '-name' THEN name END DESC,
    CASE WHEN sqlc.arg('sort')::text = 'identifier' THEN identifier END,
    CASE WHEN sqlc.arg('sort')::text = '-identifier' THEN identifier END DESC,
    CASE WHEN sqlc.arg('sort')::text = 'type' THEN type END,
    CASE WHEN sqlc.arg('sort')::text = '-type' THEN type END DESC,
    CASE WHEN sqlc.arg('sort')::text = 'criticality' THEN
        CASE criticality WHEN 'baixa' THEN 1 WHEN 'media' THEN 2 WHEN 'alta' THEN 3 WHEN 'critica' THEN 4 END
    END,
    CASE WHEN sqlc.arg('sort')::text ='-criticality' THEN
        CASE criticality WHEN 'baixa' THEN 1 WHEN 'media' THEN 2 WHEN 'alta' THEN 3 WHEN 'critica' THEN 4 END
    END DESC,
    CASE WHEN sqlc.arg('sort')::text = 'created_at' THEN created_at END,
    CASE WHEN sqlc.arg('sort')::text = '-created_at' THEN created_at END DESC,
    CASE WHEN sqlc.arg('sort')::text = 'updated_at' THEN updated_at END,
    CASE WHEN sqlc.arg('sort')::text = '-updated_at' THEN updated_at END DESC,
    id
LIMIT sqlc.arg('page_size')::int
OFFSET sqlc.arg('page_offset')::int;

-- name: CountAssets :one
SELECT count(*)
FROM assets
WHERE deleted_at IS NULL
    AND (sqlc.narg('type')::text IS NULL OR type = sqlc.narg('type'))
    AND (sqlc.narg('criticality')::text IS NULL OR criticality = sqlc.narg('criticality'))
    AND (sqlc.narg('q')::text IS NULL
        OR name ILIKE '%' || sqlc.narg('q') || '%' ESCAPE '\'
        OR identifier ILIKE '%' || sqlc.narg('q') || '%' ESCAPE '\');

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