#!/bin/sh
# Development-specific database wait and migration script

echo "[DEV] Waiting for PostgreSQL to be available..."
until nc -z -v -w30 $DB_HOST $DB_PORT; do
  echo "[DEV] Waiting for database connection..."
  sleep 2
done

echo "[DEV] PostgreSQL is up - running migrations"
# Run migrations
echo "[DEV] Running migrations from /app/db/migrations..."
if ! migrate -path /app/db/migrations -database "postgres://$DB_USER:$DB_PASSWORD@$DB_HOST:$DB_PORT/$DB_NAME?sslmode=disable" up; then
  echo "[DEV] Failed to run migrations"
  exit 1
fi

# Start the application with air for hot reloading
echo "[DEV] Starting application with air..."
exec air -c .air.toml
