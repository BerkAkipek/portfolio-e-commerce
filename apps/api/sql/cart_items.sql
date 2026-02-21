-- name: CreateCartItem :one
INSERT INTO cart_items (
  cart_id,
  product_id,
  quantity,
  price_cents_snapshot
) VALUES (
  $1, $2, $3, $4
)
RETURNING *;

-- name: GetCartItemByID :one
SELECT * FROM cart_items
WHERE id = $1
LIMIT 1;

-- name: GetCartItemByCartAndProduct :one
SELECT * FROM cart_items
WHERE cart_id = $1
  AND product_id = $2
LIMIT 1;

-- name: ListCartItemsByCartID :many
SELECT * FROM cart_items
WHERE cart_id = $1
ORDER BY created_at ASC;

-- name: UpdateCartItemQuantity :one
UPDATE cart_items
SET quantity = $2
WHERE id = $1
RETURNING *;

-- name: RemoveCartItem :exec
DELETE FROM cart_items
WHERE id = $1;

-- name: RemoveCartItemByCartAndProduct :exec
DELETE FROM cart_items
WHERE cart_id = $1
  AND product_id = $2;

