#!/bin/bash
set -e

echo "Creating databases and users..."

# Create databases
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    -- User Service Database
    CREATE DATABASE cairn_users;

    -- Recommender Service Database
    CREATE DATABASE cairn_recommender;

    -- Fetcher Service Database
    CREATE DATABASE cairn_fetcher;
EOSQL

echo "Databases created successfully!"

# Run User Service migrations
echo "Running User Service migrations..."
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname cairn_users <<-EOSQL
$(cat /docker-entrypoint-initdb.d/user-migrations/*.sql 2>/dev/null || echo "-- No user migrations found")
EOSQL

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
