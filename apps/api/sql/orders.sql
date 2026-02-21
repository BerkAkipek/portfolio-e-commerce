-- name: CreateOrder :one
INSERT INTO orders (
  user_id,
  status,
  currency,
  subtotal_cents,
  total_cents,
  shipping_name,
  shipping_address_line1,
  shipping_address_line2,
  shipping_city,
  shipping_postal_code,
  shipping_country
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: GetOrderByID :one
SELECT * FROM orders
WHERE id = $1
LIMIT 1;

-- name: ListOrdersByUserID :many
SELECT * FROM orders
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateOrderStatus :one
UPDATE orders
SET
  status = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateOrderShipping :one
UPDATE orders
SET
  shipping_name = $2,
  shipping_address_line1 = $3,
  shipping_address_line2 = $4,
  shipping_city = $5,
  shipping_postal_code = $6,
  shipping_country = $7,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

