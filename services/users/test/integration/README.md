# User Service Integration Tests

This directory contains integration tests for the Cairn User Service. These tests verify the service behavior with real PostgreSQL database and HashiCorp Vault instances.

## Overview

Integration tests validate:
- **Registration flows** (email and mobile)
- **Login flows** (email and mobile)
- **Token refresh and rotation**
- **Logout flows** (single and all sessions)
- **Account upgrade** (mobile to hybrid)
- **User management** (get, update, delete)
- **Authorization checks**
- **Concurrent operations**
- **End-to-end user lifecycles**

## Prerequisites

- Docker and Docker Compose
- Go 1.21 or higher
- Make (optional, for convenience)

## Quick Start

### Using Make (Recommended)

```bash
# Run integration tests (automatically starts/stops test environment)
make test-integration

# Or manually control the test environment
make test-integration-up      # Start test environment
make test-integration-down    # Stop and cleanup

# Run all tests (unit + integration)
make test-all
```

### Manual Setup

```bash
# 1. Start test environment
cd test/integration
docker compose -f docker-compose.test.yml up -d

# 2. Wait for services to be ready (about 10 seconds)
sleep 10

# 3. Run integration tests
go test -v -timeout 5m ./...

# 4. Stop test environment
docker compose -f docker-compose.test.yml down -v
```

## Test Environment

The integration test environment includes:

### PostgreSQL Database
- **Host**: localhost:5433
- **Database**: cairn_users_test
- **User**: cairn_test
- **Password**: test_password
- **Migrations**: Auto-applied on startup

### HashiCorp Vault
- **Address**: http://localhost:8201
- **Token**: test-root-token
- **Mode**: Development mode (in-memory)
- **JWT Keys**: Auto-generated and stored on startup

## Test Structure

```
test/integration/
├── docker-compose.test.yml    # Test environment definition
├── .env.test                  # Test environment variables
├── scripts/
│   └── init-test-vault.sh     # Vault initialization script
├── testutil/
│   └── setup.go               # Test helper utilities
└── *_test.go                  # Integration test files
    ├── auth_registration_test.go
    ├── auth_login_test.go
    ├── auth_token_test.go
    └── user_management_test.go
```

## Test Helpers

The `testutil` package provides helper functions for integration tests:

### Setup Functions
- `SetupTestEnvironment(t)` - Initialize all test dependencies
- `WaitForDatabase(t, timeout)` - Wait for database to be ready
- `WaitForVault(t, timeout)` - Wait for Vault to be ready

### Test Data Functions
- `CreateTestUser(t, email, password)` - Create email/password user
- `CreateTestMobileUser(t, deviceID)` - Create mobile-only user
- `GenerateTestTokens(t, userID, ...)` - Generate access/refresh tokens
- `RandomEmail()` - Generate random test email
- `RandomDeviceID()` - Generate random device ID

### Cleanup Functions
- `CleanupTestUser(t, userID)` - Remove user and related data
- `CleanupAllTestData(t)` - Clear all test data from database

## Running Specific Tests

```bash
# Run only registration tests
go test -v -run TestAuthService_Registration_Integration

# Run only login tests
go test -v -run TestAuthService_.*Login_Integration

# Run only token tests
go test -v -run TestAuthService_Token.*_Integration

# Run only user management tests
go test -v -run TestUserService_.*_Integration
```

## Test Coverage

Integration tests cover the following scenarios:

### Registration
- ✅ Email/password registration with validation
- ✅ Mobile device registration with device tracking
- ✅ Duplicate email/device detection
- ✅ Password strength validation
- ✅ Token generation and validation

### Login
- ✅ Email/password login with correct credentials
- ✅ Mobile device login for mobile-only accounts
- ✅ Invalid credentials handling
- ✅ Hybrid account device login prevention
- ✅ Last login timestamp updates
- ✅ Concurrent login support

### Token Management
- ✅ Token refresh with rotation
- ✅ Multiple refresh cycles
- ✅ Token reuse detection
- ✅ Grace period handling
- ✅ Token family revocation
- ✅ Device and IP tracking

### Logout
- ✅ Single session logout
- ✅ All sessions logout
- ✅ Token invalidation
- ✅ Already revoked token handling

### Account Upgrade
- ✅ Mobile to hybrid account upgrade
- ✅ Email/password addition
- ✅ Password hashing verification
- ✅ Account type validation
- ✅ Authorization checks
- ✅ Duplicate email prevention

### User Management
- ✅ Get user with authorization
- ✅ Update user email
- ✅ Delete user and cleanup
- ✅ Authorization enforcement
- ✅ Refresh token cascade deletion

### End-to-End Flows
- ✅ Complete mobile user lifecycle
- ✅ Complete email user lifecycle
- ✅ Multi-device scenarios
- ✅ Concurrent operations

## Environment Variables

Integration tests use these environment variables (defined in `.env.test`):

```bash
# Database Configuration
DB_HOST=localhost
DB_PORT=5433
DB_USER=cairn_test
DB_PASSWORD=test_password
DB_NAME=cairn_users_test
DB_SSLMODE=disable

# Vault Configuration
VAULT_ADDR=http://localhost:8201
VAULT_TOKEN=test-root-token

# JWT Configuration
JWT_ACCESS_LIFETIME=15m
JWT_REFRESH_LIFETIME=7d

# Security Configuration
BCRYPT_COST=10                    # Lower cost for faster tests
PASSWORD_MIN_LENGTH=8
PASSWORD_REQUIRE_COMPLEXITY=true
```

## Troubleshooting

### Tests fail with "database not available"
```bash
# Check if database is running
docker compose -f docker-compose.test.yml ps

# View database logs
docker compose -f docker-compose.test.yml logs test-db

# Restart database
docker compose -f docker-compose.test.yml restart test-db
```

### Tests fail with "Vault not ready"
```bash
# Check Vault status
docker compose -f docker-compose.test.yml ps

# View Vault logs
docker compose -f docker-compose.test.yml logs test-vault test-vault-init

# Restart Vault
docker compose -f docker-compose.test.yml restart test-vault
```

### Tests timeout
```bash
# Increase timeout when running tests
go test -v -timeout 10m ./...

# Or check if services are actually ready
docker compose -f docker-compose.test.yml ps
```

### Port conflicts
```bash
# If ports 5433 or 8201 are in use, modify docker-compose.test.yml:
# - "5434:5432"  # Change PostgreSQL port
# - "8202:8200"  # Change Vault port

# Then update .env.test accordingly
```

### Database migration issues
```bash
# Check if migrations ran successfully
docker compose -f docker-compose.test.yml logs test-db

# Manually run migrations
docker compose -f docker-compose.test.yml exec test-db \
  psql -U cairn_test -d cairn_users_test -c "\dt"
```

## Cleanup

```bash
# Stop and remove all test containers and volumes
make test-integration-down

# Or manually
cd test/integration
docker compose -f docker-compose.test.yml down -v

# Remove any orphaned containers
docker compose -f docker-compose.test.yml rm -f
```

## Best Practices

1. **Always cleanup test data**: Use `defer env.CleanupTestUser(t, userID)` after creating test users
2. **Use random data**: Use `RandomEmail()` and `RandomDeviceID()` to avoid conflicts
3. **Wait for dependencies**: Tests automatically wait for database and Vault to be ready
4. **Check errors**: Always verify `require.NoError(t, err)` for critical operations
5. **Test isolation**: Each test should be independent and not rely on other tests
6. **Parallel execution**: Tests can run in parallel when using `t.Parallel()` (currently disabled)

## CI/CD Integration

To run integration tests in CI/CD pipelines:

```yaml
# Example GitHub Actions workflow
jobs:
  integration-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      - name: Run integration tests
        run: make test-integration
```

## Performance

Integration tests are slower than unit tests due to:
- Docker container startup time (~10 seconds)
- Real database operations
- Real cryptographic operations (bcrypt, JWT signing)
- Network latency

Typical test execution times:
- **Setup**: ~10 seconds (Docker startup)
- **Per test**: 100-500ms
- **Total**: ~2-5 minutes for full suite

## Next Steps

After integration tests pass, proceed to:
1. **Phase 8**: Security Hardening (security review, penetration testing)
2. **Phase 10**: Documentation (API docs, deployment guides)
3. **Phase 11**: Deployment Preparation (container setup, CI/CD)

## Support

For issues or questions:
- Check the [main README](../../README.md)
- Review [implementation plan](../../todo.md)
- See [requirements](/docs/detailed_requirements/users_service_requirements.md)
