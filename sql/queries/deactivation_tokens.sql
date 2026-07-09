-- name: CreateDeactivationToken :exec
INSERT INTO deactivation_tokens (token, user_id, expires_at)
VALUES ($1, $2, $3);

-- name: GetDeactivationToken :one
SELECT user_id, expires_at FROM deactivation_tokens
WHERE token = $1;

-- name: DeleteDeactivationToken :exec
DELETE FROM deactivation_tokens
WHERE token = $1;
