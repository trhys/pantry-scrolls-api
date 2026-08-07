#!/bin/sh
set -e

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

DB="$DB_URL" ./categoriesMigration

echo "migration completed"
