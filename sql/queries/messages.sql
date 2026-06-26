-- name: AddMessage :exec
INSERT INTO messages (user_email, message)
VALUES (
    $1,
    $2
);

-- name: GetAllMessages :many
SELECT * FROM messages
WHERE status != 'archived';

-- name: GetMessagesWithTag :many
SELECT * FROM messages 
WHERE status = $1;