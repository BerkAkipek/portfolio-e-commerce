-- name: CreateCategory :one
INSERT INTO categories (
  name,
  slug,
  parent_id
) VALUES (
  $1, $2, $3
)
RETURNING *;

-- name: GetCategoryByID :one
SELECT * FROM categories
WHERE id = $1
LIMIT 1;

-- name: GetCategoryBySlug :one
SELECT * FROM categories
WHERE LOWER(slug) = LOWER($1)
LIMIT 1;

-- name: ListRootCategories :many
SELECT * FROM categories
WHERE parent_id IS NULL
ORDER BY name ASC;

-- name: ListChildCategories :many
SELECT * FROM categories
WHERE parent_id = sqlc.arg(parent_id)::uuid
ORDER BY name ASC;

-- name: ListCategories :many
SELECT * FROM categories
ORDER BY name ASC
LIMIT $1 OFFSET $2;
