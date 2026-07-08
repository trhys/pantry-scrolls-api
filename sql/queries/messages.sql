-- name: AddMessage :exec
INSERT INTO messages (id, user_email, message)
VALUES (
    gen_random_uuid(),
    $1,
    $2
);

-- name: GetAllMessages :many
SELECT * FROM messages
WHERE status != 'archived';

-- name: GetMessagesWithTag :many
SELECT * FROM messages 
WHERE status = $1;

-- name: GetMessageById :one
SELECT * FROM messages 
WHERE id = $1;

-- name: SetMessageStatus :exec
UPDATE messages
SET status = $1
WHERE id = $2;
