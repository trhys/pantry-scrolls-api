-- name: GetServerState :one
SELECT active, message FROM server_state
LIMIT 1;

-- name: SetServerState :exec
UPDATE server_state
SET active = $1, message = $2
WHERE id = 1;
