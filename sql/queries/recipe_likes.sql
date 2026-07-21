-- name: LikeRecipe :exec
INSERT INTO recipe_likes (user_id, recipe_id)
VALUES (
  $1,
  $2
);

-- name: GetLikes :one
SELECT COUNT(*) FROM recipe_likes
WHERE recipe_id = $1;

-- name: UnlikeRecipe :exec
DELETE FROM recipe_likes
WHERE (user_id, recipe_id) = ($1, $2);
