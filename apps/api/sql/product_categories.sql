-- name: AddProductCategory :exec
INSERT INTO product_categories (
  product_id,
  category_id
) VALUES (
  $1, $2
)
ON CONFLICT DO NOTHING;

-- name: RemoveProductCategory :exec
DELETE FROM product_categories
WHERE product_id = $1
  AND category_id = $2;

-- name: ListCategoriesByProductID :many
SELECT c.*
FROM categories c
INNER JOIN product_categories pc ON pc.category_id = c.id
WHERE pc.product_id = $1
ORDER BY c.name ASC;

-- name: ListProductsByCategoryID :many
SELECT p.*
FROM products p
INNER JOIN product_categories pc ON pc.product_id = p.id
WHERE pc.category_id = $1
ORDER BY p.created_at DESC;

