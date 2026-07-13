-- +goose Up
-- server_state holds a single row representing the current server maintenance state.
CREATE TABLE server_state (
  id SERIAL PRIMARY KEY,
  active BOOLEAN NOT NULL DEFAULT FALSE,
  message TEXT NOT NULL DEFAULT '',
  CONSTRAINT single_row CHECK (id = 1)
  );

-- +goose Down
DROP TABLE server_state;
