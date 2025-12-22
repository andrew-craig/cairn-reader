#!/bin/bash

# Script to manually run database migrations
# Usage: ./scripts/run-migration.sh <migration-file>

set -e

MIGRATION_FILE=${1:-""}
DB_CONTAINER="cairn-explore-postgres-1"

if [ -z "$MIGRATION_FILE" ]; then
    echo "Usage: ./scripts/run-migration.sh <migration-file>"
    echo "Example: ./scripts/run-migration.sh recommender/migrations/002_add_feeds_table.sql"
    exit 1
fi

if [ ! -f "$MIGRATION_FILE" ]; then
    echo "Error: Migration file not found: $MIGRATION_FILE"
    exit 1
fi

echo "Running migration: $MIGRATION_FILE"

# Check if container is running
if ! docker ps | grep -q "$DB_CONTAINER"; then
    echo "Error: PostgreSQL container is not running"
    echo "Start services with: docker-compose up -d"
    exit 1
fi

# Copy migration file to container and execute
docker cp "$MIGRATION_FILE" "$DB_CONTAINER:/tmp/migration.sql"
docker exec -i "$DB_CONTAINER" psql -U cairn -d cairn_db -f /tmp/migration.sql

echo "Migration completed successfully!"
