-- +goose Up
CREATE TABLE messages (
  id UUID PRIMARY KEY,
  user_email TEXT NOT NULL,
  message TEXT NOT NULL,
  resolved BOOLEAN NOT NULL DEFAULT FALSE,
  status TEXT NOT NULL DEFAULT 'unread',
  FOREIGN KEY (user_email) REFERENCES users(email)
);

-- +goose Down
DROP TABLE messages;
