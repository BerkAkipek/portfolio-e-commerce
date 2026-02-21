-- name: CreateOrderItem :one
INSERT INTO order_items (
  order_id,
  product_id,
  product_name_snapshot,
  price_cents_snapshot,
  quantity
) VALUES (
  $1, $2, $3, $4, $5
)
RETURNING *;

-- name: GetOrderItemByID :one
SELECT * FROM order_items
WHERE id = $1
LIMIT 1;

-- name: ListOrderItemsByOrderID :many
SELECT * FROM order_items
WHERE order_id = $1
ORDER BY created_at ASC;

