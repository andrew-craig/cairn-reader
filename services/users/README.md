# Cairn User Service

The Cairn User Service is responsible for managing user access to the Cairn read-it-later service, including user registration, authentication, and account management.

## Features

- User registration with email/password or mobile device ID
- Stateless JWT authentication with RS256 signing
- Refresh token management with automatic rotation
- Mobile device authentication via Expo device ID
- Account upgrade from device-only to email/password
- Secure secrets management with HashiCorp Vault
- Authorization middleware ensuring users can only access their own data

## Project Structure

```
services/users/
├── cmd/
│   ├── user-service/        # Application entrypoint
│   └── migrate/             # Migration CLI (used by `make migrate-*`)
├── internal/                # Private application code
│   ├── auth/               # JWT signing, refresh tokens, passwords, Vault client
│   ├── config/             # Configuration management
│   ├── database/           # Database connection, migrations and repositories
│   ├── handlers/           # HTTP handlers and router
│   ├── models/             # Data models
│   ├── services/           # Business logic
│   └── validation/         # Input validation
├── api/openapi.yaml        # OpenAPI specification
├── migrations/             # Database migrations
├── selfhost/               # Init wrappers for the consolidated self-hosted binary
├── test/                   # Integration and security tests
├── go.mod                  # Go module definition
└── README.md              # This file
```

CORS, rate limiting, security headers, recovery and JWT validation come from the shared `pkg/middleware` and `pkg/auth` modules at the repo root.

## Prerequisites

- Go 1.25 or higher (see `go.mod`)
- PostgreSQL (the dev compose stack uses 18)
- HashiCorp Vault (required: JWT signing keys are read from it)

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/andrew-craig/cairn-reader.git
cd cairn-reader/services/users
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Set up environment variables

```bash
# No .env.example ships in this directory. Set the variables listed in
# CLAUDE.md under "Environment Variables" (a .env file is loaded if present).
```

### 4. Set up PostgreSQL database

```bash
createdb cairn_users
```

### 5. Run database migrations

```bash
# Apply all pending migrations
make migrate-up

# Check current migration version
make migrate-version

# Rollback the last migration (if needed)
make migrate-down
```

### 6. Set up HashiCorp Vault (Development)

For local development, you can run Vault in dev mode:

```bash
vault server -dev
```

Store JWT keys in Vault:

```bash
# Generate RSA key pair
openssl genrsa -out private.pem 2048
openssl rsa -in private.pem -pubout -out public.pem

# Store in Vault
vault kv put secret/jwt/private-key key=@private.pem
vault kv put secret/jwt/public-key key=@public.pem
```

### 7. Run the service

```bash
go run cmd/user-service/main.go
```

The service will start on port 8080 (or the `PORT` you set). The dev docker-compose stack publishes it on host port 8082.

## API Endpoints

### Health & Status

- `GET /health/live` - Liveness check
- `GET /health/ready` - Readiness check (includes database and Vault connectivity)

### Authentication

- `POST /api/v1/auth/register` - Create new user account with email/password
- `POST /api/v1/auth/register/mobile` - Create new mobile-only account using Expo device ID
- `POST /api/v1/auth/login` - Validate credentials and return tokens
- `POST /api/v1/auth/login/mobile` - Authenticate using Expo device ID
- `POST /api/v1/auth/refresh` - Exchange refresh token for new access token
- `POST /api/v1/auth/logout` - Revoke specific refresh token
- `POST /api/v1/auth/logout-all` - Revoke all refresh tokens for a user (authenticated)
- `POST /api/v1/auth/resend-verification` - Resend the email verification message (authenticated)
- `POST` / `GET /api/v1/auth/verify-email` - Verify an email address with a token

### User Management

- `GET /api/v1/user/{user_id}` - Retrieve user profile (authenticated)
- `PATCH /api/v1/user/{user_id}` - Update user profile (authenticated)
- `POST /api/v1/user/{user_id}/upgrade` - Add email and password to mobile-only account
- `PUT /api/v1/user/{user_id}/password` - Change password (authenticated)
- `DELETE /api/v1/user/{user_id}` - Delete user account (authenticated)

Request/response schemas are in [api/openapi.yaml](api/openapi.yaml). `/api/v1/*` requires HTTPS (TLS or `X-Forwarded-Proto: https`), so add `-H "X-Forwarded-Proto: https"` to direct `curl` calls. Default password rules require upper, lower, digit and special characters.

**Example API Calls:**

```bash
# Health checks
curl http://localhost:8082/health/live
curl http://localhost:8082/health/ready

# Register user
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"securepass123"}' \
  http://localhost:8082/api/v1/auth/register

# Login
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"email":"user@example.com","password":"securepass123"}' \
  http://localhost:8082/api/v1/auth/login

# Refresh token
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{"refresh_token":"<refresh_token>"}' \
  http://localhost:8082/api/v1/auth/refresh

# Get user profile (requires auth)
curl -H "Authorization: Bearer <JWT>" \
  http://localhost:8082/api/v1/user/{user_id}

# Update user profile (requires auth)
curl -X PATCH \
  -H "Authorization: Bearer <JWT>" \
  -H "Content-Type: application/json" \
  -d '{"email":"newemail@example.com"}' \
  http://localhost:8082/api/v1/user/{user_id}

# Delete user account (requires auth)
curl -X DELETE \
  -H "Authorization: Bearer <JWT>" \
  http://localhost:8082/api/v1/user/{user_id}
```

## Configuration

See [CLAUDE.md](CLAUDE.md#environment-variables) for all configuration options and defaults.

Key configuration areas:
- Server settings (port, environment)
- Database connection
- HashiCorp Vault integration
- JWT token lifetimes
- Security settings (bcrypt cost, password requirements, rate limiting)

## Security

- Passwords are hashed using bcrypt (cost 12 by default, 10-14 allowed)
- JWT tokens signed with RS256 (2048-bit RSA keys)
- Refresh tokens are hashed before database storage
- All secrets managed through HashiCorp Vault in production
- Rate limiting on authentication endpoints
- HTTPS required on all `/api/v1/*` routes
- Authorization middleware ensures users can only access their own data

## Development

### Running Tests

```bash
make test        # unit tests; see CLAUDE.md for integration/security targets
```

### Running with Live Reload

```bash
# Install air for live reloading
go install github.com/air-verse/air@latest

# Run with air
air
```

## Deployment

See [requirements](/docs/detailed_requirements/users_service_requirements.md) for detailed deployment requirements and considerations.

## Documentation

- [Requirements](/docs/detailed_requirements/users_service_requirements.md) - Detailed service requirements
- [CLAUDE.md](CLAUDE.md) - Architecture, configuration and development guide

## License

[Your License Here]
