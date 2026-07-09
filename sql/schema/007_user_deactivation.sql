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

CREATE INDEX idx_deactivation_tokens_expires_at ON deactivation_tokens(expires_at);

-- +goose Down
DROP INDEX idx_deactivation_tokens_expires_at;
DROP TABLE deactivation_tokens;
ALTER TABLE users DROP COLUMN deactivated_at;
