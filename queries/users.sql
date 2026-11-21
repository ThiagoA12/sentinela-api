-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: CreateUser :one
INSERT INTO users (name, email, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING id, name, email, password_hash, role, active, failed_logins,
    locked_until, created_at, updated_at, version;
