-- name: CreateShoppingList :one
INSERT INTO shopping_lists (id, name, created_at, updated_at, user_id)
VALUES (
	gen_random_uuid(),
	$1,
	NOW(),
	NOW(),
	$2
) RETURNING *;

-- name: GetShoppingList :one
SELECT * FROM shopping_lists
WHERE id = $1;

-- name: GetUserLists :many
SELECT * FROM shopping_lists
WHERE user_id = $1;

-- name: GetListOwner :one
SELECT name, user_id FROM shopping_lists
WHERE id = $1;

-- name: PrintList :many
SELECT 
	ingredients.name, 
	recipe_ingredients.ingredient_id, 
	(recipe_ingredients.quantity * shopping_list_recipes.quantity)::REAL AS quantity, 
	recipe_ingredients.unit, 
	conversions.to_unit, 
	conversions.ratio 
FROM recipe_ingredients
INNER JOIN shopping_list_recipes ON shopping_list_recipes.recipe_id = recipe_ingredients.recipe_id
INNER JOIN conversions ON conversions.ingredient_id = recipe_ingredients.ingredient_id AND conversions.from_unit = recipe_ingredients.unit
INNER JOIN ingredients ON ingredients.id = recipe_ingredients.ingredient_id
WHERE shopping_list_recipes.shopping_list_id = $1;

-- name: DeleteShoppingList :exec
DELETE FROM shopping_lists
WHERE id = $1;
