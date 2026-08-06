-- name: CreateIngredient :one
INSERT INTO ingredients (id, name, image_key, created_at, updated_at)
VALUES (
	gen_random_uuid(),
	$1,
	$2,
	NOW(),
	NOW()
) RETURNING *;

-- name: GetIngredients :many
SELECT id, name FROM ingredients;

-- name: GetIngredientName :one
SELECT name FROM ingredients
WHERE id = $1;

-- name: GetIngredientFromName :one
SELECT id FROM ingredients
WHERE name = $1;

-- name: AddCategory :exec
UPDATE ingredients
SET category = $1
WHERE name = $2;

-- name: GetCategory :many
SELECT * FROM ingredients
WHERE category = $1;
