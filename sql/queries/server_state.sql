-- name: GetServerState :one
SELECT active, message FROM server_state
LIMIT 1;
