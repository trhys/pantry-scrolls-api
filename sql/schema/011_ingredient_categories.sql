-- +goose Up
ALTER TABLE ingredients
ADD COLUMN category TEXT NOT NULL;

-- +goose Down
ALTER TABLE ingredients
DROP COLUMN category;
