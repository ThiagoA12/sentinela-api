-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (name, email, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING id, name, email, password_hash, role, active, failed_logins,
    locked_until, created_at, updated_at, version;

-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, role, active, failed_logins,
    locked_until, created_at, updated_at, version
FROM users
WHERE lower(email) = lower(sqlc.arg('email'));

-- name: RecordLoginFailure :exec
UPDATE users
SET failed_logins = CASE WHEN failed_logins + 1 >= sqlc.arg('max_failures')::smallint
        THEN 0 ELSE failed_logins + 1 END,
    locked_until = CASE WHEN failed_logins + 1 >= sqlc.arg('max_failures')::smallint
        THEN sqlc.arg('locked_until')::timestamptz ELSE locked_until END
WHERE id = sqlc.arg('id');

-- name: ResetLoginFailures :exec
UPDATE users
SET failed_logins = 0, locked_until = NULL
WHERE id = $1;
