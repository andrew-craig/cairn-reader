#!/bin/bash
set -e

# Self-hosted database initialization.
# Creates all 6 logical databases owned by a single user.
# Runs inside the PostgreSQL container on first startup.

echo "=== Cairn Self-Hosted Database Initialization ==="
echo "Creating databases for all services..."

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
    CREATE DATABASE cairn_users;
    CREATE DATABASE cairn_recommender;
    CREATE DATABASE cairn_fetcher;
    CREATE DATABASE content_service;
    CREATE DATABASE rss_fetcher_service;
    CREATE DATABASE ingest_email;
EOSQL

echo "=== All databases created successfully ==="
