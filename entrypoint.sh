#!/bin/sh
set -e

echo "Running schema migrations..."
cd /app/knex-migrations
npx knex migrate:latest --env docker

echo "Migrating SQLite data into Postgres..."
node migrate-sqlite-to-postgres.js

echo "Starting Go app..."
exec /app/app
