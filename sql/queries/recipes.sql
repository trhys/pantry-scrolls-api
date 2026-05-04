-- name: CreateRecipe :one
INSERT INTO recipes (id, title, author, created_at, updated_at, user_id, description, image_key, instructions)
VALUES(
	gen_random_uuid(),
	$1,
	$2,
	NOW(),
	NOW(),
	$3,
	$4,
	$5,
	$6
)
RETURNING *;

-- name: GetRecipe :one
SELECT * FROM recipes
WHERE id = $1;

-- name: GetRecipeList :many
SELECT * FROM recipes
ORDER BY created_at DESC
LIMIT 10;

-- name: GetUsersRecipes :many
SELECT * FROM recipes
WHERE user_id = $1
ORDER BY created_at DESC;
