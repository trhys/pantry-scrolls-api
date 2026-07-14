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

-- name: UpdateRecipe :one
UPDATE recipes SET 
	title = $1,
	description = $2,
	image_key = $3,
	instructions = $4,
	updated_at = NOW()
WHERE id = $5
RETURNING *;

-- name: GetRecipe :one
SELECT * FROM recipes
WHERE id = $1;

-- name: GetRecipeList :many
SELECT * FROM recipes
ORDER BY created_at DESC
LIMIT 10;

-- name: GetRecipesFromQuery :many
SELECT * FROM recipes
WHERE LOWER(title) LIKE '%' || $1::text || '%';

-- name: GetUsersRecipes :many
SELECT * FROM recipes
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetRecipeOwner :one
SELECT user_id FROM recipes
WHERE id = $1;

-- name: GetRecipeImageKey :one
SELECT image_key FROM recipes
WHERE id = $1;

-- name: DeleteRecipe :exec
DELETE FROM recipes
WHERE id = $1;

-- name: CheckIfSeeded :one
SELECT id FROM recipes
WHERE description = $1;

-- name: GetTotalRecipes :one
SELECT COUNT(*) FROM recipes;