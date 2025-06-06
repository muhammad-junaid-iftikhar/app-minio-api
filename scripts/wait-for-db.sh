#!/bin/sh
# Production database wait script

set -e

host="$1"
shift
cmd="$@"

echo "[PROD] Waiting for PostgreSQL to be available..."
until PGPASSWORD=$DB_PASSWORD psql -h "$host" -U "$DB_USER" -d "$DB_NAME" -c '\q'; do
  >&2 echo "[PROD] PostgreSQL is unavailable - sleeping"
  sleep 2
done

>&2 echo "[PROD] PostgreSQL is up - executing command"
exec $cmd
