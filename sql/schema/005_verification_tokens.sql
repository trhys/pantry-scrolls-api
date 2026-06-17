-- +goose Up
CREATE TABLE verification_tokens (
  email TEXT NOT NULL,
  token TEXT NOT NULL,
  expires_at TIMESTAMP NOT NULL,
  FOREIGN KEY (email) REFERENCES users(email),
  PRIMARY KEY (email, token)
);

-- +goose Down
DROP TABLE verification_tokens;
