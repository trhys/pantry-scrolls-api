-- name: AddRecipeTags :exec
INSERT INTO recipe_tags (recipe_id, tag)
SELECT sqlc.arg(recipe_id), tag
FROM UNNEST(sqlc.arg(tags)::text[]) AS tag
ON CONFLICT (recipe_id, tag) DO NOTHING;

-- name: RemoveRecipeTag :exec
DELETE FROM recipe_tags
WHERE recipe_id = $1 AND tag = $2;

-- name: ResetRecipeTags :exec
DELETE FROM recipe_tags
WHERE recipe_id = $1;
