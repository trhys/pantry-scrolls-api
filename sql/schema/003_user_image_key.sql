-- +goose Up
ALTER TABLE users
ADD COLUMN image_key TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users
DROP COLUMN image_key;
