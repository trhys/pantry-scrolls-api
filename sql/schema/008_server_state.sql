-- +goose Up
CREATE TABLE server_state (
  id SERIAL PRIMARY KEY,
  active BOOLEAN NOT NULL DEFAULT FALSE,
  message TEXT NOT NULL DEFAULT ''
  );

-- +goose Down
DROP TABLE server_state;
