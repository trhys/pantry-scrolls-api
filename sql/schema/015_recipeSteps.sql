-- +goose Up
ALTER TABLE recipes
ADD COLUMN instructions TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE recipes
DROP COLUMN instructions;
