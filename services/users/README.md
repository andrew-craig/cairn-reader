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
cairn/services/users/
├── cmd/
│   └── user-service/        # Application entrypoint
│       └── main.go
├── internal/                # Private application code
│   ├── auth/               # JWT and token management
│   ├── config/             # Configuration management
│   ├── database/           # Database connection and repositories
│   ├── handlers/           # HTTP request handlers
│   ├── middleware/         # HTTP middleware (auth, rate limiting, etc.)
│   └── models/             # Data models
├── pkg/                    # Public libraries (shared auth package)
│   └── auth/               # Shared JWT validation library
├── migrations/             # Database migrations
├── .env.example            # Example environment configuration
├── go.mod                  # Go module definition
└── README.md              # This file
```

## Prerequisites

- Go 1.21 or higher
- PostgreSQL 14 or higher
- HashiCorp Vault (for production)

## Getting Started

### 1. Clone the repository

```bash
git clone https://github.com/andrew-craig/cairn.git
cd cairn/services/users
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Set up environment variables

```bash
cp .env.example .env
# Edit .env with your configuration
```

### 4. Set up PostgreSQL database

```bash
createdb cairn_users
```

### 5. Run database migrations

```bash
# TODO: Add migration instructions once migration tool is set up
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
vault kv put secret/jwt/private-key value=@private.pem
vault kv put secret/jwt/public-key value=@public.pem
```

### 7. Run the service

```bash
go run cmd/user-service/main.go
```

The service will start on port 8080 (or the port specified in your .env file).

## API Endpoints

### Authentication

- `POST /auth/register` - Create new user account with email/password
- `POST /auth/register/mobile` - Create new mobile-only account using Expo device ID
- `POST /auth/login` - Validate credentials and return tokens
- `POST /auth/login/mobile` - Authenticate using Expo device ID
- `POST /auth/refresh` - Exchange refresh token for new access token
- `POST /auth/logout` - Revoke specific refresh token
- `POST /auth/logout-all` - Revoke all refresh tokens for a user

### User Management

- `GET /users/{id}` - Retrieve user profile (authenticated)
- `PATCH /users/{id}` - Update user profile (authenticated)
- `POST /users/{id}/upgrade` - Add email and password to mobile-only account
- `DELETE /users/{id}` - Delete user account (authenticated)

### Health & Status

- `GET /health` - Basic health check
- `GET /ready` - Readiness check (includes database and Vault connectivity)

## Configuration

See [.env.example](.env.example) for all available configuration options.

Key configuration areas:
- Server settings (port, environment)
- Database connection
- HashiCorp Vault integration
- JWT token lifetimes
- Security settings (bcrypt cost, password requirements, rate limiting)

## Security

- Passwords are hashed using bcrypt with cost factor 12+
- JWT tokens signed with RS256 (2048-bit RSA keys)
- Refresh tokens are hashed before database storage
- All secrets managed through HashiCorp Vault in production
- Rate limiting on authentication endpoints
- HTTPS required in production
- Authorization middleware ensures users can only access their own data

## Development

### Running Tests

```bash
go test ./...
```

### Running with Live Reload

```bash
# Install air for live reloading
go install github.com/cosmtrek/air@latest

# Run with air
air
```

## Deployment

See [requirements.md](requirements.md) for detailed deployment requirements and considerations.

## Documentation

- [Requirements](requirements.md) - Detailed service requirements
- [Implementation Plan](todo.md) - Phased implementation checklist

## License

[Your License Here]
