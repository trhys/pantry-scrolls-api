-- name: AddRecipeTags :exec
INSERT INTO recipe_tags (recipe_id, tag)
VALUES (
  $1,
  SELECT * FROM UNNEST(@tags::text[]))
);

-- name: RemoveRecipeTag :exec
DELETE FROM recipe_tags
WHERE recipe_id = $1 AND tag = $2;

-- name: ResetRecipeTags :exec
DELETE FROM recipe_tags
WHERE recipe_id = $1;
