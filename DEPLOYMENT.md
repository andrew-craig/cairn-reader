# Cairn Deployment Guide

This guide covers deploying all Cairn backend services using Docker Compose.

## Quick Start

### Development Deployment

```bash
cd infrastructure/docker
docker compose up --build
```

That's it! All services will start automatically with:
- ✅ Vault for secrets management
- ✅ PostgreSQL with 3 databases
- ✅ JWT keys auto-generated
- ✅ Database migrations auto-run
- ✅ All services connected and healthy

### Service Endpoints

After deployment, services are available at:

| Service | URL | Purpose |
|---------|-----|---------|
| User Service | http://localhost:8082 | Authentication, user management |
| Recommender | http://localhost:8081 | Article recommendations, voting |
| Fetcher | http://localhost:8080 | RSS feed fetching |
| Vault UI | http://localhost:8200/ui | Secrets management (dev token: `dev-token`) |
| PostgreSQL | localhost:5432 | Database (user: `cairn`, password: `cairn_password`) |

## Architecture

```
┌─────────────┐
│   Vault     │ ← Manages JWT keys
│   :8200     │
└──────┬──────┘
       │
       ├──────────────────────────────┐
       │                              │
┌──────▼──────┐  ┌───────────┐  ┌────▼──────────┐
│User Service │  │  Fetcher  │  │  Recommender  │
│   :8082     │  │   :8080   │──│     :8081     │
└──────┬──────┘  └─────┬─────┘  └───────┬───────┘
       │               │                │
       └───────────────┼────────────────┘
                       │
              ┌────────▼─────────┐
              │   PostgreSQL     │
              │      :5432       │
              ├──────────────────┤
              │ cairn_users      │  ← User accounts
              │ cairn_recommender│  ← Articles, votes
              │ cairn_fetcher    │  ← RSS feeds
              └──────────────────┘
```

## What Gets Deployed

### Services

1. **Vault** (dev mode)
   - Auto-generates RSA key pair for JWT
   - Stores keys securely
   - No manual setup required

2. **User Service**
   - User registration and authentication
   - JWT token issuance
   - Account management

3. **Recommender Service**
   - Stores articles from RSS feeds
   - Implements recommendation algorithm
   - Handles upvote/downvote
   - Requires JWT authentication

4. **Fetcher Service**
   - Fetches 1 RSS feed per minute
   - Syncs feed list daily from Kagi Small Web
   - Sends articles to Recommender

5. **PostgreSQL**
   - Single instance, three databases
   - Automatic migration execution
   - Persistent data storage

## Testing the Deployment

### 1. Check Service Health

```bash
# All services
docker compose ps

# Individual health endpoints
curl http://localhost:8082/health  # User Service
curl http://localhost:8081/health  # Recommender
curl http://localhost:8080/health  # Fetcher
```

### 2. Register a User

```bash
curl -X POST http://localhost:8082/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "TestPassword123!"
  }'
```

Expected response:
```json
{
  "user": {
    "id": "...",
    "email": "test@example.com"
  },
  "access_token": "eyJ...",
  "refresh_token": "..."
}
```

### 3. Get Recommendations

```bash
# Save the access_token from registration
TOKEN="your_access_token_here"

curl http://localhost:8081/api/v1/recommendations/ \
  -H "Authorization: Bearer $TOKEN"
```

### 4. Trigger Manual Feed Fetch

```bash
curl -X POST http://localhost:8080/fetch
```

## Database Access

### Connect to PostgreSQL

```bash
# Using docker compose exec
docker compose exec postgres psql -U cairn

# List databases
docker compose exec postgres psql -U cairn -c '\l'

# Connect to specific database
docker compose exec postgres psql -U cairn -d cairn_users
```

### Database Contents

**cairn_users:**
- users
- refresh_tokens

**cairn_recommender:**
- users (activity tracking)
- articles
- user_articles (read status)
- votes
- recommendations

**cairn_fetcher:**
- feeds
- fetch_history

## Managing the Deployment

### View Logs

```bash
# All services
docker compose logs -f

# Specific service
docker compose logs -f recommender
docker compose logs -f user-service
docker compose logs -f fetcher
```

### Restart Services

```bash
# Restart all
docker compose restart

# Restart specific service
docker compose restart recommender
```

### Stop Services

```bash
# Stop but keep data
docker compose down

# Stop and remove all data
docker compose down -v
```

### Rebuild After Code Changes

```bash
# Rebuild specific service
docker compose up --build recommender

# Rebuild all services
docker compose up --build
```

## File Structure

```
infrastructure/docker/
├── docker-compose.yml          # Service definitions
├── scripts/
│   ├── init-vault.sh          # Auto-generates JWT keys
│   └── init-postgres.sh       # Creates databases, runs migrations
└── README.md                   # Detailed deployment docs
```

## Environment Variables

All configured in [docker-compose.yml](infrastructure/docker/docker-compose.yml).

Key settings:
- `VAULT_DEV_ROOT_TOKEN_ID=dev-token` (⚠️ dev only)
- `POSTGRES_PASSWORD=cairn_password` (⚠️ change for production)
- `JWT_ACCESS_LIFETIME=15m`
- `JWT_REFRESH_LIFETIME=7d`
- `FETCH_INTERVAL=60` (seconds, 1 feed/minute)
- `ARTICLE_RETENTION_DAYS=90`

## Troubleshooting

### Services won't start

```bash
# Check service status
docker compose ps

# Check logs for errors
docker compose logs
```

Common issues:
- Port conflicts (change ports in docker-compose.yml)
- Docker not running
- Insufficient memory (increase Docker memory limit)

### Database connection errors

```bash
# Verify database is healthy
docker compose ps postgres

# Check migration logs
docker compose logs postgres

# Verify databases exist
docker compose exec postgres psql -U cairn -l
```

### Vault errors

```bash
# Check Vault logs
docker compose logs vault
docker compose logs vault-init

# Verify keys were stored
docker compose exec vault vault kv get secret/jwt/public-key
```

### Reset everything

```bash
# Nuclear option - removes all data
docker compose down -v
docker compose up --build
```

## Production Considerations

⚠️ **This setup is for DEVELOPMENT only!**

For production:

1. **Vault**
   - Use production Vault with proper storage backend
   - Real authentication (not dev token)
   - Enable TLS
   - Set up unsealing keys

2. **Database**
   - Use strong passwords
   - Enable SSL/TLS
   - Set up automated backups
   - Consider managed PostgreSQL (AWS RDS, etc.)

3. **Secrets**
   - Use Docker secrets or Kubernetes secrets
   - Never commit passwords to git
   - Rotate credentials regularly

4. **Networking**
   - Add reverse proxy (nginx/traefik)
   - Use SSL certificates (Let's Encrypt)
   - Configure firewall rules

5. **Monitoring**
   - Add Prometheus metrics
   - Configure log aggregation
   - Set up alerting
   - Health check monitoring

6. **Scaling**
   - Multiple service instances
   - Load balancing
   - Database replication
   - Connection pooling

## Next Steps

After deployment:

1. **Integrate Mobile App**: Configure app to use deployed APIs
2. **Add Content**: RSS feeds will auto-sync from Kagi Small Web
3. **Monitor**: Check logs and health endpoints
4. **Customize**: Modify environment variables as needed

## Support

For detailed documentation:
- [infrastructure/docker/README.md](infrastructure/docker/README.md) - Comprehensive deployment guide
- [CLAUDE.md](CLAUDE.md) - Development and architecture guide
- [services/users/README.md](services/users/README.md) - User Service docs
- [services/explore/README.md](services/explore/README.md) - Explore Service docs
