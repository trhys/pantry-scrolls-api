-- name: CreateVerification :exec
INSERT INTO verification_tokens (email, token, expires_at)
VALUES (
  $1,
  $2,
  NOW() + interval '30 minutes'
);

-- name: GetVerification :one
SELECT email, expires_at FROM verification_tokens
WHERE token = $1;
