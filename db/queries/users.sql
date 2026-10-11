-- name: CreateUser :one
INSERT INTO users (
    email,
    password_hash,
    full_name
) VALUES (
    $1, $2, $3
)
RETURNING id, email, password_hash, full_name, is_active, created_at, updated_at;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, full_name, is_active, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, email, password_hash, full_name, is_active, created_at, updated_at
FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET full_name = COALESCE($2, full_name),
    updated_at = NOW()
WHERE id = $1
RETURNING id, email, password_hash, full_name, is_active, created_at, updated_at;
