-- name: CreateUser :one
INSERT INTO users (
  email,
  password_hash,
  phone,
  full_name,
  role
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE LOWER(email) = LOWER($1)
LIMIT 1;

-- name: GetUserByPhone :one
SELECT * FROM users
WHERE phone = sqlc.arg(phone)::text
LIMIT 1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateUserProfile :one
UPDATE users
SET
  full_name = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserRole :one
UPDATE users
SET
  role = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetUserPasswordHash :one
UPDATE users
SET
  password_hash = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateUserPhone :one
UPDATE users
SET
  phone = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: VerifyUser :one
UPDATE users
SET
  is_verified = TRUE,
  updated_at = NOW()
WHERE id = $1
RETURNING *;
