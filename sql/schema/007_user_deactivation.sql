-- +goose Up
ALTER TABLE users ADD COLUMN deactivated_at TIMESTAMP;

CREATE TABLE deactivation_tokens (
  token TEXT PRIMARY KEY,
  user_id UUID NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  CONSTRAINT fk_users
  FOREIGN KEY (user_id)
  REFERENCES users(id)
  ON DELETE CASCADE
);

-- +goose Down
DROP TABLE deactivation_tokens;
ALTER TABLE users DROP COLUMN deactivated_at;
