#!/bin/sh
set -e

echo "Running schema migrations..."
cd /app/knex-migrations

# Behøves ikke at blive lagt i secrets
export DB_HOST=${DB_HOST:-postgres}
export DB_PORT=${DB_PORT:-5432}

export DB_USER="$(cat /run/secrets/db_user)"
export DB_PASSWORD="$(cat /run/secrets/db_password)"
export DB_NAME="$(cat /run/secrets/db_name)"
export ADMIN_USER="$(cat /run/secrets/admin_user)"
export ADMIN_EMAIL="$(cat /run/secrets/admin_email)"
export ADMIN_PASS_HASH="$(cat /run/secrets/admin_pass_hash)"

npx knex migrate:latest --env docker

echo "Migrating SQLite data into Postgres..."
node migrate-sqlite-to-postgres.js

echo "Starting Go app..."
exec /app/app