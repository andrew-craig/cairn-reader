# Integration Tests - Quick Start Guide

## TL;DR

```bash
# Navigate to user service directory
cd services/users

# Run all integration tests (auto start/stop environment)
make test-integration

# That's it! ✨
```

## What Just Happened?

The `make test-integration` command:
1. ✅ Started PostgreSQL test database (port 5433)
2. ✅ Started HashiCorp Vault (port 8201)
3. ✅ Generated JWT keys and stored them in Vault
4. ✅ Ran all database migrations
5. ✅ Executed 40+ integration tests
6. ✅ Cleaned up and stopped all containers

## Common Commands

```bash
# Run only integration tests
make test-integration

# Run only unit tests
make test-unit
# or just
make test

# Run ALL tests (unit + integration)
make test-all

# Get unit test coverage report
make test-coverage
# Opens coverage.html in your browser
```

## Manual Control

If you want to keep the test environment running:

```bash
# Start test environment
make test-integration-up

# Run tests manually (multiple times if needed)
cd test/integration
go test -v ./...

# Run specific test
go test -v -run TestAuthService_Registration

# Stop environment when done
make test-integration-down
```

## Check Test Environment

```bash
# Start environment
make test-integration-up

# Check status
docker-compose -f test/integration/docker-compose.test.yml ps

# View logs
docker-compose -f test/integration/docker-compose.test.yml logs

# Test database connection
psql -h localhost -p 5433 -U cairn_test -d cairn_users_test
# Password: test_password

# Test Vault connection
export VAULT_ADDR=http://localhost:8201
export VAULT_TOKEN=test-root-token
vault status
vault kv get secret/jwt/public-key
```

## Troubleshooting

### Tests fail immediately
```bash
# Increase wait time in test
# Edit test files to use longer timeout:
testutil.WaitForDatabase(t, 60*time.Second)  # Was 30s
```

### Port already in use
```bash
# Check what's using the port
lsof -i :5433  # PostgreSQL
lsof -i :8201  # Vault

# Stop conflicting services or change ports in docker-compose.test.yml
```

### Database connection refused
```bash
# Check if database is running
docker-compose -f test/integration/docker-compose.test.yml ps

# Restart database
docker-compose -f test/integration/docker-compose.test.yml restart test-db

# View database logs
docker-compose -f test/integration/docker-compose.test.yml logs test-db
```

### Vault not ready
```bash
# View Vault logs
docker-compose -f test/integration/docker-compose.test.yml logs test-vault test-vault-init

# Restart Vault
docker-compose -f test/integration/docker-compose.test.yml restart test-vault
```

### Clean start
```bash
# Complete cleanup and restart
make test-integration-down
docker system prune -f
make test-integration-up
```

## Test Results

Expected output:
```
=== RUN   TestAuthService_Registration_Integration
=== RUN   TestAuthService_Registration_Integration/Register_with_email_and_password
=== RUN   TestAuthService_Registration_Integration/Register_with_duplicate_email
...
PASS
ok      github.com/cairn-app/cairn-reader/services/users/test/integration    12.345s
```

All tests should PASS ✅

## Quick Reference

| Command | What It Does |
|---------|--------------|
| `make test-integration` | Run all integration tests (auto start/stop) |
| `make test-integration-up` | Start test environment only |
| `make test-integration-down` | Stop test environment and cleanup |
| `make test-unit` | Run unit tests only |
| `make test-all` | Run both unit and integration tests |
| `make test-coverage` | Generate unit test coverage report |

## Environment Details

When running, you'll have:
- **PostgreSQL**: localhost:5433 (user: cairn_test, db: cairn_users_test)
- **Vault**: http://localhost:8201 (token: test-root-token)
- **Test Duration**: ~2-5 minutes for full suite
- **Network**: Isolated Docker network (cairn-test-network)

## Need More Info?

- Full documentation: [README.md](README.md)
- Implementation details: [IMPLEMENTATION_SUMMARY.md](IMPLEMENTATION_SUMMARY.md)
- Main service docs: [../../README.md](../../README.md)

---

**Have Questions?** Check the main [README.md](README.md) or [todo.md](../../todo.md) for details.
