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
RETURNING id;

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
SELECT recipes.*, COUNT(recipe_likes.user_id) AS likes FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
WHERE recipes.id = $1
GROUP BY recipes.id;

-- name: GetAuthedRecipe :one
SELECT recipes.*,
	COUNT(recipe_likes.user_id) AS likes,
	EXISTS(
		SELECT 1 FROM recipe_likes AS requester_likes
		WHERE requester_likes.recipe_id = recipes.id
		AND requester_likes.user_id = sqlc.arg(user_id)
	) AS liked
FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
WHERE recipes.id = sqlc.arg(id)
GROUP BY recipes.id;

-- name: GetRecipeList :many
SELECT recipes.*, COUNT(recipe_likes.user_id) AS likes FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
GROUP BY recipes.id
ORDER BY likes DESC
LIMIT 10;

-- name: GetAuthedRecipeList :many
SELECT recipes.*,
	COUNT(recipe_likes.user_id) AS likes,
	EXISTS(
		SELECT 1 FROM recipe_likes AS requester_likes
		WHERE requester_likes.recipe_id = recipes.id
		AND requester_likes.user_id = sqlc.arg(user_id)
	) AS liked
FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
GROUP BY recipes.id
ORDER BY likes DESC
LIMIT 10;

-- name: GetRecipesFromQuery :many
SELECT recipes.*, COUNT(recipe_likes.user_id) AS likes FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
WHERE LOWER(title) LIKE '%' || $1::text || '%'
GROUP BY recipes.id
LIMIT 50;

-- name: GetAuthedRecipesFromQuery :many
SELECT recipes.*,
	COUNT(recipe_likes.user_id) AS likes,
	EXISTS(
		SELECT 1 FROM recipe_likes AS requester_likes
		WHERE requester_likes.recipe_id = recipes.id
		AND requester_likes.user_id = sqlc.arg(user_id)
	) AS liked
FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
WHERE LOWER(title) LIKE '%' || sqlc.arg(query)::text || '%'
GROUP BY recipes.id
LIMIT 50;

-- name: GetRecipesFromNilQuery :many
SELECT recipes.*, COUNT(recipe_likes.user_id) AS likes FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
GROUP BY recipes.id
ORDER BY created_at DESC
LIMIT 50;

-- name: GetAuthedRecipesFromNilQuery :many
SELECT recipes.*,
	COUNT(recipe_likes.user_id) AS likes,
	EXISTS(
		SELECT 1 FROM recipe_likes AS requester_likes
		WHERE requester_likes.recipe_id = recipes.id
		AND requester_likes.user_id = sqlc.arg(user_id)
	) AS liked
FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
GROUP BY recipes.id
ORDER BY created_at DESC
LIMIT 50;

-- name: GetUsersRecipes :many
SELECT recipes.*, COUNT(recipe_likes.user_id) AS likes FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
WHERE recipes.user_id = $1
GROUP BY recipes.id
ORDER BY created_at DESC;

-- name: GetAuthedUsersRecipes :many
SELECT recipes.*,
	COUNT(recipe_likes.user_id) AS likes,
	EXISTS(
		SELECT 1 FROM recipe_likes AS requester_likes
		WHERE requester_likes.recipe_id = recipes.id
		AND requester_likes.user_id = sqlc.arg(requester_id)
	) AS liked
FROM recipes
LEFT JOIN recipe_likes ON recipe_likes.recipe_id = recipes.id
WHERE recipes.user_id = sqlc.arg(user_id)
GROUP BY recipes.id
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
