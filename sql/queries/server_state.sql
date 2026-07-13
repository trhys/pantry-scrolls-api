-- name: GetServerState :one
SELECT active, message FROM server_state
ORDER BY id ASC
LIMIT 1;
