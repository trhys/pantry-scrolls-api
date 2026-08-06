-- +goose Up
ALTER TABLE ingredients
ADD COLUMN category TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE ingredients
DROP COLUMN category;
