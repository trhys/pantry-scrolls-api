-- +goose Up
CREATE TABLE messages (
  id UUID PRIMARY KEY,
  user_email TEXT NOT NULL,
  message TEXT NOT NULL,
  resolved BOOLEAN NOT NULL DEFAULT FALSE,
  status TEXT NOT NULL DEFAULT 'unread'
  );

-- +goose Down
DROP TABLE messages;
