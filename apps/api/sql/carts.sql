-- name: CreateCart :one
INSERT INTO carts (
  user_id,
  session_id,
  status
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetCartByID :one
SELECT * FROM carts
WHERE id = $1
LIMIT 1;

-- name: GetActiveCartByUserID :one
SELECT * FROM carts
WHERE user_id = sqlc.arg(user_id)::uuid
  AND status = 'active'
ORDER BY created_at DESC
LIMIT 1;

-- name: GetActiveCartBySessionID :one
SELECT * FROM carts
WHERE session_id = sqlc.arg(session_id)::text
  AND status = 'active'
ORDER BY created_at DESC
LIMIT 1;

-- name: ListCartsByUserID :many
SELECT * FROM carts
WHERE user_id = sqlc.arg(user_id)::uuid
ORDER BY created_at DESC
LIMIT sqlc.arg(limit_rows) OFFSET sqlc.arg(offset_rows);

-- name: UpdateCartStatus :one
UPDATE carts
SET
  status = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: AttachCartToUser :one
UPDATE carts
SET
  user_id = sqlc.arg(user_id)::uuid,
  session_id = NULL,
  updated_at = NOW()
WHERE id = sqlc.arg(id)
RETURNING *;
