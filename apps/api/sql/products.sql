-- name: CreateProduct :one
INSERT INTO products (
  name,
  slug,
  description,
  price_cents,
  currency,
  stock,
  is_active,
  sku,
  weight_grams,
  image_url
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
)
RETURNING *;

-- name: GetProductByID :one
SELECT * FROM products
WHERE id = $1
LIMIT 1;

-- name: GetProductBySlug :one
SELECT * FROM products
WHERE LOWER(slug) = LOWER($1)
LIMIT 1;

-- name: ListActiveProducts :many
SELECT * FROM products
WHERE is_active = TRUE
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateProduct :one
UPDATE products
SET
  name = $2,
  slug = $3,
  description = $4,
  price_cents = $5,
  currency = $6,
  stock = $7,
  is_active = $8,
  sku = $9,
  weight_grams = $10,
  image_url = $11,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateProductStock :one
UPDATE products
SET
  stock = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetProductActiveStatus :one
UPDATE products
SET
  is_active = $2,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

