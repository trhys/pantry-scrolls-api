-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_pw, name)
VALUES (
	gen_random_uuid(),
	NOW(),
	NOW(),
	$1,
	$2,
	$3
) RETURNING id, created_at, email, name;

-- name: GetUserHash :one
SELECT id, email, name, hashed_pw, image_key FROM users
WHERE email = $1;

-- name: GetUser :one
SELECT id, created_at, updated_at, email, name, image_key FROM USERS
WHERE id = $1;

-- name: GetName :one
SELECT name FROM users
WHERE id = $1;

-- name: CheckAdmin :one
SELECT admin FROM users
WHERE id = $1;

-- name: MakeAdmin :exec
UPDATE users
SET admin = true
WHERE id = $1;

-- name: RefreshUser :one
SELECT id, name, email, image_key FROM users
WHERE id = $1;

-- name: SetUserImageKey :exec
UPDATE users
SET image_key = $2
WHERE id = $1;

-- name: GetUserImageKey :one
SELECT image_key FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, name, email FROM users
WHERE email = $1;

-- name: UpdateUser :exec
UPDATE users
SET name = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdatePasswordHash :exec
UPDATE users 
SET hashed_pw = $2, updated_at = NOW()
WHERE email = $1;

-- name: VerifyEmail :exec
UPDATE users
SET is_verified = TRUE
WHERE email = $1;

-- name: GetUserEmail :one
SELECT email FROM users
WHERE id = $1;