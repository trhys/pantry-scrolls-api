-- name: AddMessage :exec
INSERT INTO messages (user_email, message)
VALUES (
    $1,
    $2
);