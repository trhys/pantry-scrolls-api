#!/bin/sh
set -e

# Start postgres using the official entrypoint in the background
docker-entrypoint.sh postgres &
PG_PID=$!

# Wait for postgres to accept connections
echo "Waiting for postgres to be ready..."
until pg_isready -U "$POSTGRES_USER" -d "$POSTGRES_DB"; do
    sleep 1
done
echo "Postgres is ready."

DB_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@localhost:5432/${POSTGRES_DB}?sslmode=disable"

# Run schema migrations
echo "Running goose migrations..."
./goose -dir sql/schema postgres "$DB_URL" up

# Run seed
echo "Running db seed..."
DB="$DB_URL" ./dbinit

echo "DB init complete."

# Keep postgres running
wait $PG_PID
