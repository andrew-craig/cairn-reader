# Fetcher Database Migrations

This directory contains SQL migration files for the fetcher service's PostgreSQL database.

## Migrations

### 001_init_schema.sql
Initial schema for feed management:
- **feeds**: Stores RSS feed sources with health tracking
- **fetch_history**: Records each fetch attempt for monitoring

## Running Migrations

Migrations are automatically applied when the `fetcher_db` PostgreSQL container starts up, thanks to the volume mount in docker-compose.yml:

```yaml
volumes:
  - ./fetcher/migrations:/docker-entrypoint-initdb.d
```

### Manual Migration

To run migrations manually on an existing database:

```bash
# Connect to the database
docker exec -it <fetcher_db_container> psql -U fetcher -d fetcher_db

# Or from host if you have psql
psql -h localhost -p 5433 -U fetcher -d fetcher_db

# Then run:
\i /docker-entrypoint-initdb.d/001_init_schema.sql
```

### Reset Database

To start fresh and re-run all migrations:

```bash
docker-compose down
docker volume rm cairn-explore_fetcher_postgres_data
docker-compose up --build
```

## Schema Overview

### feeds table
- `id`: Primary key
- `url`: Unique feed URL
- `title`: Feed title (populated after first fetch)
- `description`: Feed description
- `last_fetched_at`: Timestamp of last successful fetch (NULL if never fetched)
- `consecutive_failures`: Counter for health tracking
- `enabled`: Whether feed is active (auto-disabled after 10 failures)
- `created_at`, `updated_at`: Timestamps

### fetch_history table
- `id`: Primary key
- `feed_id`: Foreign key to feeds table
- `fetch_started_at`: When fetch started
- `fetch_completed_at`: When fetch completed
- `success`: Whether fetch succeeded
- `articles_found`: Number of articles in feed
- `articles_sent`: Number successfully sent to recommender
- `error_message`: Error details if failed
- `created_at`: Timestamp

## Indexes

Performance indexes are created for:
- Finding next feed to fetch (by `last_fetched_at` and `enabled`)
- Querying fetch history by feed_id
- Sorting fetch history by date
