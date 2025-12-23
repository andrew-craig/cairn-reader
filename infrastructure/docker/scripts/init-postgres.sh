#!/bin/bash
set -e

echo "Creating databases and users..."

# Create databases - connect to postgres database
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
    -- User Service Database
    CREATE DATABASE cairn_users;

    -- Recommender Service Database
    CREATE DATABASE cairn_recommender;

    -- Fetcher Service Database
    CREATE DATABASE cairn_fetcher;
EOSQL

echo "Databases created successfully!"

# Note: User Service migrations are handled by the application itself using golang-migrate
# Running them here would create a dirty state in the schema_migrations table

# Run Recommender migrations
echo "Running Recommender Service migrations..."
for f in /docker-entrypoint-initdb.d/recommender-migrations/*.sql; do
    if [ -f "$f" ]; then
        echo "Applying migration: $f"
        psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname cairn_recommender -f "$f"
    fi
done

# Run Fetcher migrations
echo "Running Fetcher Service migrations..."
for f in /docker-entrypoint-initdb.d/fetcher-migrations/*.sql; do
    if [ -f "$f" ]; then
        echo "Applying migration: $f"
        psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname cairn_fetcher -f "$f"
    fi
done

echo "All migrations completed successfully!"
