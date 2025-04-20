#!/bin/sh
set -e

echo "Running schema migrations..."
cd /app/knex-migrations

export DB_HOST=${DB_HOST:-postgres}
export DB_PORT=${DB_PORT:-5432}
export DB_USER=${DB_USER:-youruser}
export DB_PASSWORD=${DB_PASSWORD:-yourpassword}
export DB_NAME=${DB_NAME:-whoknows}
export ADMIN_USER=${ADMIN_USER:-admin}
export ADMIN_EMAIL=${ADMIN_EMAIL:-keamonk1@stud.kea.dk}
export ADMIN_PASS_HASH=${ADMIN_PASS_HASH:-5f4dcc3b5aa765d61d8327deb882cf99}

npx knex migrate:latest --env docker

echo "Migrating SQLite data into Postgres..."
node migrate-sqlite-to-postgres.js

echo "Starting Go app..."
exec /app/app