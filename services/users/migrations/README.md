# Database Migrations

This directory contains database migration files for the Cairn User Service.

## Migration Files

Migrations are managed using [golang-migrate](https://github.com/golang-migrate/migrate) and follow a sequential numbering scheme:

- `000001_create_users_table` - Creates the users table with support for email/password and mobile device authentication
- `000002_create_refresh_tokens_table` - Creates the refresh_tokens table for JWT refresh token management
- `000003_drop_refresh_tokens_trigger` - Drops a broken `last_used_at` trigger on refresh_tokens
- `000004_add_login_lockout` - Adds `failed_login_attempts`, `locked_until` and `last_failed_login_at` to users
- `000005_add_email_verification` - Adds `users.email_verified` and the `email_verification_tokens` table
- `000006_create_password_reset_tokens_table` - Creates `password_reset_tokens` (never used)
- `000007_add_revoked_at_to_refresh_tokens` - Adds `refresh_tokens.revoked_at` so revoked tokens are kept until expiry for reuse detection
- `000008_drop_password_reset_tokens_table` - Drops `password_reset_tokens` (password reset was removed)

Each migration has a matching `.down.sql`; `embed.go` embeds the files for the self-hosted binary.

## Running Migrations

### Using Make (Recommended)

```bash
# Run all pending migrations
make migrate-up

# Rollback the last migration
make migrate-down

# Check current migration version
make migrate-version
```

### Using the Migration CLI

```bash
# Run all pending migrations
go run cmd/migrate/main.go -command=up

# Rollback the last migration
go run cmd/migrate/main.go -command=down

# Check current migration version
go run cmd/migrate/main.go -command=version

# Specify custom migrations path
go run cmd/migrate/main.go -command=up -path=./migrations

# Use custom .env file
go run cmd/migrate/main.go -command=up -env=.env.production
```

## Database Schema

### Users Table

Stores user account information with support for three account types:
- **Mobile-only**: `expo_device_id` is set, `email` and `password_hash` are NULL
- **Email-only**: `email` and `password_hash` are set, `expo_device_id` is NULL
- **Hybrid**: All three fields are set (upgraded from mobile-only)

Fields:
- `id` - UUID primary key
- `email` - Unique email address (nullable)
- `password_hash` - Bcrypt password hash (nullable)
- `expo_device_id` - Expo Application Installation ID (nullable)
- `created_at` - Timestamp of account creation
- `updated_at` - Timestamp of last update (auto-updated via trigger)
- `last_login_at` - Timestamp of last successful login
- `email_verified` - Whether the email has been verified (migration 000005)
- `failed_login_attempts`, `locked_until`, `last_failed_login_at` - Login lockout tracking (migration 000004)

Indexes:
- `idx_users_email` - Partial index on email (where email IS NOT NULL)
- `idx_users_expo_device_id` - Partial index on expo_device_id (where expo_device_id IS NOT NULL)
- `idx_users_created_at` - Index on created_at for query performance

### Refresh Tokens Table

Stores refresh tokens for JWT authentication with rotation support.

Fields:
- `id` - UUID primary key
- `user_id` - Foreign key to users table (cascading delete)
- `token_hash` - SHA-256 hash of the refresh token (unique)
- `expires_at` - Token expiration timestamp
- `created_at` - Token creation timestamp
- `last_used_at` - Last time token was used for refresh
- `device_info` - User agent or device information
- `ip_address` - IP address from which token was created
- `token_family` - UUID for tracking token rotation chains
- `revoked_at` - When the token was rotated away or logged out; NULL means active (migration 000007)

Indexes:
- `idx_refresh_tokens_user_id` - Index on user_id for user token lookups
- `idx_refresh_tokens_token_hash` - Index on token_hash for token validation
- `idx_refresh_tokens_expires_at` - Index on expires_at for cleanup operations
- `idx_refresh_tokens_token_family` - Partial index on token_family for rotation tracking
- `idx_refresh_tokens_revoked_at` - Partial index on revoked_at (where revoked_at IS NOT NULL)

### Email Verification Tokens Table

Backs the email verification endpoints (migration 000005).

Fields: `id` (UUID primary key), `user_id` (foreign key to users, cascading delete), `token_hash` (SHA-256 hex digest, unique), `expires_at`, `created_at`.

## Database Setup

### PostgreSQL Requirements

- PostgreSQL 12 or higher
- `gen_random_uuid()` function (available by default in PostgreSQL 13+)

### Creating the Database

```bash
# Create database and user
make db-create

# Or manually:
psql -U postgres
CREATE DATABASE cairn_users;
CREATE USER cairn_user WITH PASSWORD 'your_secure_password_here';
GRANT ALL PRIVILEGES ON DATABASE cairn_users TO cairn_user;
```

### Configuration

Database configuration is loaded from environment variables (see [CLAUDE.md](../CLAUDE.md#environment-variables)); a `.env` file is also read if present (`-env` flag):

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=cairn_user
DB_PASSWORD=your_secure_password_here
DB_NAME=cairn_users
DB_SSLMODE=disable  # Use "require" or "verify-full" in production
```

## Migration Best Practices

1. **Never modify existing migrations** - Once a migration is committed and deployed, create a new migration for changes
2. **Always test rollbacks** - Ensure down migrations work correctly before deploying
3. **Use transactions** - Migrations run in transactions by default for safety
4. **Index carefully** - Use partial indexes where appropriate to optimize query performance
5. **Document constraints** - Use CHECK constraints to enforce business rules at the database level

## Troubleshooting

### Dirty Database State

If a migration fails partway through, the database may be in a "dirty" state:

```bash
# Check migration version and dirty state
make migrate-version

# Manual intervention may be required
# Fix the issue, then force the version:
# psql -U cairn_user -d cairn_users
# UPDATE schema_migrations SET dirty = false;
```

### Connection Issues

Ensure your database is running and connection parameters are correct:

```bash
# Test database connection
psql -h localhost -U cairn_user -d cairn_users

# Check PostgreSQL is running
pg_isready -h localhost -p 5432
```
