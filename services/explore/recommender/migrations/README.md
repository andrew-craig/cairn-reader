# Database Migrations

This directory contains SQL migration files for the Cairn Explore recommender service database.

## Migration Files

- `001_init.sql` - Initial schema with users, articles, and user_articles tables
- `002_add_feed_id_to_articles.sql` - Adds feed_id column to articles
- `003_fetcher_schema_updates.sql` - Schema updates for voting, deduplication, and renamed columns
- `004_voting_and_recommendations.sql` - Voting and recommendation tracking tables (Phase 1.1)

## Running Migrations

### Automatic (Docker first-time setup)

When starting the PostgreSQL container for the first time, all migrations in this directory are automatically executed via Docker's `/docker-entrypoint-initdb.d` mechanism.

```bash
# Fresh start - migrations run automatically
docker-compose up --build
```

### Manual (for existing databases)

If the database already exists and you need to apply a new migration:

```bash
# Option 1: Use the migration script
./scripts/run-migration.sh recommender/migrations/002_add_feeds_table.sql

# Option 2: Direct psql execution (requires postgres container running)
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_db -f /docker-entrypoint-initdb.d/002_add_feeds_table.sql

# Option 3: Copy and execute
docker cp recommender/migrations/002_add_feeds_table.sql cairn-explore-postgres-1:/tmp/migration.sql
docker exec -i cairn-explore-postgres-1 psql -U cairn -d cairn_db -f /tmp/migration.sql
```

### Reset Database (apply all migrations fresh)

```bash
# Stop services
docker-compose down

# Remove database volume
docker volume rm cairn-explore_postgres_data

# Start services (migrations run automatically on fresh database)
docker-compose up --build
```

## Migration Guidelines

1. **Sequential Naming**: Use sequential numbering (001, 002, 003, etc.)
2. **Idempotent**: Use `IF NOT EXISTS` clauses where possible
3. **Description**: Include clear comments describing what the migration does
4. **Testing**: Test migrations on a development database before production
5. **Rollback**: Consider creating corresponding rollback migrations if needed

## Current Schema

See individual migration files for complete schema details.

### Tables (as of 004_voting_and_recommendations.sql)

- **users** - User accounts (id SERIAL, user_id TEXT external identifier)
- **articles** - RSS feed articles with voting and recommendation tracking (upvotes, downvotes, recommends, deleted)
- **user_articles** - Tracks read status per user
- **article_categories** - Normalized article category relationships
- **votes** - Individual vote tracking per user (upvote/downvote with UNIQUE constraint)
- **recommendations** - Tracks which articles recommended to which users

## Future Enhancements

Consider using a migration tool like:
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [goose](https://github.com/pressly/goose)
- [dbmate](https://github.com/amacneil/dbmate)

These tools provide version tracking, rollbacks, and better migration management.
