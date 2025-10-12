-- name: CreateUser :one
INSERT INTO users (id, name, created_at, updated_at)
VALUES ($1,
        $2,
        $3,
        $4)
RETURNING *;

-- name: GetUser :one
SELECT *
FROM users
WHERE name = $1;

-- name: ClearUsers :exec
TRUNCATE users CASCADE;

-- name: GetUsers :many
SELECT *
FROM users;
