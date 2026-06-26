-- name: AddMessage :exec
INSERT INTO messages (email, message)
VALUES (
    $1,
    $2
);