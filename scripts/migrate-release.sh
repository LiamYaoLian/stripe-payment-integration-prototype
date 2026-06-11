#!/usr/bin/env bash
set -euo pipefail

: "${DATABASE_URL:?DATABASE_URL is required}"

MIGRATE_IMAGE="${MIGRATE_IMAGE:-migrate/migrate:v4.18.2}"
MIGRATIONS_DIR="${MIGRATIONS_DIR:-$(cd "$(dirname "$0")/../backend/migrations" && pwd)}"

echo "Running database migrations..."
docker run --rm \
  -v "${MIGRATIONS_DIR}:/migrations:ro" \
  "${MIGRATE_IMAGE}" \
  -path=/migrations \
  -database "${DATABASE_URL}" \
  up

echo "Migrations complete."
