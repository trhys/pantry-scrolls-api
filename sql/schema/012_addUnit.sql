-- +goose Up
ALTER TABLE shopping_list_ingredients
ADD COLUMN units TEXT NOT NULL REFERENCES units(name);

-- +goose Down
ALTER TABLE shopping_list_ingredients
DROP COLUMN IF EXISTS units;
