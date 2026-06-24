-- +goose Up
CREATE TABLE messages (
  user_email TEXT NOT NULL,
  message TEXT NOT NULL,
  resolved BOOLEAN NOT NULL DEFAULT FALSE,
  status TEXT NOT NULL DEFAULT 'unread',
  FOREIGN KEY (user_email) REFERENCES users(email),
  PRIMARY KEY (user_email, message)
);

-- +goose Down
DROP TABLE messages;
