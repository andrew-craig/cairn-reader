# Infrastructure Docker Compose Migration Guide

## Database Architecture Change

The infrastructure docker-compose.yml has been updated to use **separate PostgreSQL instances** for each service instead of a single instance with multiple databases.

### Before (Single Instance)
```
postgres (single instance)
├── cairn_users (database)
├── cairn_recommender (database)
└── cairn_fetcher (database)
```

### After (Separate Instances)
```
users-db:5432 → cairn_users
recommender-db:5433 → cairn_recommender
fetcher-db:5434 → cairn_fetcher
```

## Benefits

1. **Microservices Isolation**: Each service owns its own database instance
2. **Independent Scaling**: Databases can be scaled independently
3. **Resource Isolation**: Better resource management and fault isolation
4. **Consistency**: Matches pattern used in explore and read services
5. **Simpler Deployment**: No shared init scripts or dependencies

## Migration Steps

### 1. Stop Existing Services
```bash
cd infrastructure/docker
docker compose down
```

### 2. Clean Up Old Data (if starting fresh)
```bash
docker volume rm docker_postgres_data
```

**⚠️ Warning**: This will delete all existing data. Skip this step if you want to preserve data.

### 3. Update Environment File
Copy the new environment template:
```bash
cp .env.example .env
```

Edit `.env` with your actual values:
```bash
# HashiCorp Vault Configuration
VAULT_DEV_ROOT_TOKEN_ID=dev-token-123

# User Service Database
POSTGRES_USER_USERS=cairn_user
POSTGRES_PASSWORD_USERS=secure_password_1
POSTGRES_DB_USERS=cairn_users

# Recommender Service Database
POSTGRES_USER_RECOMMENDER=cairn_recommender
POSTGRES_PASSWORD_RECOMMENDER=secure_password_2
POSTGRES_DB_RECOMMENDER=cairn_recommender

# Fetcher Service Database
POSTGRES_USER_FETCHER=cairn_fetcher
POSTGRES_PASSWORD_FETCHER=secure_password_3
POSTGRES_DB_FETCHER=cairn_fetcher
```

### 4. Start Services
```bash
docker compose up --build
```

## Data Migration (if needed)

If you need to migrate existing data from the old single-instance setup:

1. Export data from old instance:
```bash
# Start old instance
docker compose up postgres

# Export each database
docker exec <postgres_container_id> pg_dump -U cairn cairn_users > users_backup.sql
docker exec <postgres_container_id> pg_dump -U cairn cairn_recommender > recommender_backup.sql
docker exec <postgres_container_id> pg_dump -U cairn cairn_fetcher > fetcher_backup.sql
```

2. Import data to new instances:
```bash
# Start new instances
docker compose up users-db recommender-db fetcher-db

# Import to each database
docker exec -i <users_db_container_id> psql -U cairn_user cairn_users < users_backup.sql
docker exec -i <recommender_db_container_id> psql -U cairn_recommender cairn_recommender < recommender_backup.sql
docker exec -i <fetcher_db_container_id> psql -U cairn_fetcher cairn_fetcher < fetcher_backup.sql
```

## Verification

Check that all services are healthy:
```bash
docker compose ps
```

All services should show "healthy" status.

Test each service:
```bash
# User Service
curl http://localhost:8082/health

# Recommender Service
curl http://localhost:8081/health

# Fetcher Service
curl http://localhost:8080/health
```

## Troubleshooting

### Port Conflicts
If you see port binding errors, check that ports 5432, 5433, and 5434 are available:
```bash
lsof -i :5432
lsof -i :5433
lsof -i :5434
```

### Database Connection Issues
Check database logs:
```bash
docker compose logs users-db
docker compose logs recommender-db
docker compose logs fetcher-db
```

### Service Health Checks Failing
View service logs:
```bash
docker compose logs user-service
docker compose logs recommender
docker compose logs fetcher
```

## Removed Files

The following file is no longer used:
- `scripts/init-postgres.sh` (database creation now handled by individual PostgreSQL containers)

Migrations are now run automatically when each database container starts for the first time.
