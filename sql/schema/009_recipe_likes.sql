-- +goose Up
CREATE TABLE recipe_likes (
  user_id UUID NOT NULL,
  recipe_id UUID NOT NULL,
  FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  FOREIGN KEY (recipe_id) REFERENCES recipes(id) ON DELETE CASCADE,
  PRIMARY KEY (user_id, recipe_id)
);

-- +goose Down
DROP TABLE recipe_likes;
