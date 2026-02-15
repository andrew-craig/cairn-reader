#!/bin/bash
set -e

# This script runs inside the single PostgreSQL container on first startup.
# It creates all the logical databases and their dedicated users so each
# service still connects to its own database with its own credentials.
#
# Environment variables are passed from the docker compose configuration.

echo "=== Cairn Database Initialization ==="
echo "Creating databases and users for all services..."

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
    -- User Service
    CREATE USER ${POSTGRES_USER_USERS} WITH PASSWORD '${POSTGRES_PASSWORD_USERS}';
    CREATE DATABASE ${POSTGRES_DB_USERS} OWNER ${POSTGRES_USER_USERS};

    -- Explore Recommender
    CREATE USER ${POSTGRES_USER_RECOMMENDER} WITH PASSWORD '${POSTGRES_PASSWORD_RECOMMENDER}';
    CREATE DATABASE ${POSTGRES_DB_RECOMMENDER} OWNER ${POSTGRES_USER_RECOMMENDER};

    -- Explore Fetcher
    CREATE USER ${POSTGRES_USER_FETCHER} WITH PASSWORD '${POSTGRES_PASSWORD_FETCHER}';
    CREATE DATABASE ${POSTGRES_DB_FETCHER} OWNER ${POSTGRES_USER_FETCHER};

    -- Content Service
    CREATE USER ${POSTGRES_USER_CONTENT} WITH PASSWORD '${POSTGRES_PASSWORD_CONTENT}';
    CREATE DATABASE ${POSTGRES_DB_CONTENT} OWNER ${POSTGRES_USER_CONTENT};

    -- Ingest RSS
    CREATE USER ${POSTGRES_USER_RSS} WITH PASSWORD '${POSTGRES_PASSWORD_RSS}';
    CREATE DATABASE ${POSTGRES_DB_RSS} OWNER ${POSTGRES_USER_RSS};
EOSQL

echo "=== All databases and users created successfully ==="
