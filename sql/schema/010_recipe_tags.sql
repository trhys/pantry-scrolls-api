-- +goose Up
CREATE TABLE recipe_tags (
  recipe_id UUID NOT NULL,
  tag TEXT NOT NULL,
  FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE,
  PRIMARY KEY (recipe_id, tag)
);

-- +goose Down
DROP TABLE recipe_tags;
