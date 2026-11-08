#!/bin/sh
# Reset the Stash database — wipe every user-data row but keep the schema.
# Works on self-hosted (Docker) and cloud (Neon) Postgres alike.
#
# Usage:
#   cd server
#   ./scripts/reset-db.sh              # uses .env.local
#   ./scripts/reset-db.sh .env.cloud   # uses .env.cloud
#   ./scripts/reset-db.sh .env.production

set -e

ENV_FILE="${1:-.env.local}"

if [ ! -f "$ENV_FILE" ]; then
  echo "=== Env file $ENV_FILE not found"
  exit 1
fi

# Source the env file to get DB_* variables
set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

DB_HOST_VALUE="${DB_HOST:-localhost}"
DB_PORT_VALUE="${DB_HOST_PORT:-5432}"
DB_URL="postgres://${DB_USERNAME}:${DB_PASSWORD}@${DB_HOST_VALUE}:${DB_PORT_VALUE}/${DB_NAME}?sslmode=${DB_SSLMODE:-disable}"

echo "=== Resetting database ${DB_NAME} on ${DB_HOST_VALUE}:${DB_PORT_VALUE}"

# TRUNCATE every user-data table in one transaction.
# CASCADE handles FK constraints.
# The events table has an append-only trigger that blocks TRUNCATE;
# disable it for the reset and re-enable it afterward.
# schema_migrations is left alone so migrate doesn't try to re-apply.
psql "$DB_URL" <<SQL
BEGIN;
  ALTER TABLE events DISABLE TRIGGER events_append_only;
  TRUNCATE TABLE events, checkpoints, item_tags, tags, items RESTART IDENTITY CASCADE;
  ALTER TABLE events ENABLE TRIGGER events_append_only;
COMMIT;
SQL

echo "=== Database ${DB_NAME} reset"