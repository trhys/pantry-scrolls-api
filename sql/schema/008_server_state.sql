-- +goose Up
-- server_state holds a single row representing the current server maintenance state.
CREATE TABLE server_state (
  id SERIAL PRIMARY KEY,
  active BOOLEAN NOT NULL DEFAULT FALSE,
  message TEXT NOT NULL DEFAULT '',
  CONSTRAINT single_row CHECK (id = 1)
  );

INSERT INTO server_state (id, active, message)
VALUES (1, FALSE, '');

-- +goose Down
DROP TABLE server_state;
