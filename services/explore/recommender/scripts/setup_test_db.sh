#!/bin/bash
set -e

# Test database configuration
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-cairn}
DB_PASSWORD=${DB_PASSWORD:-cairn_password}
TEST_DB_NAME=${TEST_DB_NAME:-cairn_test_db}
CONTAINER_NAME=${CONTAINER_NAME:-cairn-explore-postgres-1}

echo "Setting up test database: $TEST_DB_NAME"

# Check if we should use Docker
USE_DOCKER=false
if ! command -v psql &> /dev/null; then
    if docker ps | grep -q $CONTAINER_NAME; then
        USE_DOCKER=true
        echo "Using Docker container: $CONTAINER_NAME"
    else
        echo "Error: psql not found and Docker container not running"
        exit 1
    fi
fi

# Drop and recreate test database
if [ "$USE_DOCKER" = true ]; then
    docker exec -i $CONTAINER_NAME psql -U $DB_USER -d postgres <<EOF
DROP DATABASE IF EXISTS $TEST_DB_NAME;
CREATE DATABASE $TEST_DB_NAME;
EOF
else
    PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d postgres <<EOF
DROP DATABASE IF EXISTS $TEST_DB_NAME;
CREATE DATABASE $TEST_DB_NAME;
EOF
fi

echo "Running migrations on test database..."

# Run all migrations
MIGRATIONS_DIR="$(dirname "$0")/../migrations"

for migration in "$MIGRATIONS_DIR"/*.sql; do
    if [ -f "$migration" ]; then
        echo "Running migration: $(basename $migration)"
        if [ "$USE_DOCKER" = true ]; then
            docker exec -i $CONTAINER_NAME psql -U $DB_USER -d $TEST_DB_NAME < "$migration"
        else
            PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $TEST_DB_NAME -f "$migration"
        fi
    fi
done

echo "Test database setup complete!"
