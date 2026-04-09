# Requirements - Cairn User Service

## Overview

Cairn is a mobile read-it-later service that helps users read their longform content and discover new content.

Cairn is implemented as a React Native mobile app and a modular backend implemented in Go.

The Cairn User Service is responsible for managing users access to the service, and managing their user account. It implements stateless JWT authentication with refresh token capability.

## Service Responsibilities
* User registration and account management
* Credential validation and authentication
* JWT access token issuance and management
* Refresh token issuance, validation, and revocation
* User profile data storage and retrieval
* Mobile device authentication via Expo Identification Key
* Account upgrade from device-only to email/password authentication

## Core Endpoints
Authentication
* POST /auth/register - Create new user account with email/password, return access and refresh tokens
* POST /auth/register/mobile - Create new mobile-only account using Expo device ID, return access and refresh tokens
* POST /auth/login - Validate credentials, return access and refresh tokens
* POST /auth/login/mobile - Authenticate using Expo device ID, return access and refresh tokens
* POST /auth/refresh - Exchange valid refresh token for new access token
* POST /auth/logout - Revoke specific refresh token
* POST /auth/logout-all - Revoke all refresh tokens for a user

User Management
* GET /users/{id} - Retrieve user profile (authenticated)
* PATCH /users/{id} - Update user profile (authenticated)
* POST /users/{id}/upgrade - Add email and password to mobile-only account (authenticated)
* DELETE /users/{id} - Delete user account (authenticated)

Authorization
* Users can only access their own user ID
* Requests to access other user IDs return 403 Forbidden
* User ID extracted from validated JWT token claims

## Token Specifications
Access Token (JWT)
* Type: JSON Web Token (JWT)
* Lifetime: 60 minutes (configurable)
* Storage: Client memory
* Claims: user_id, issued_at, expires_at
* Validation: Stateless - validated by consuming services without database lookup
* Algorithm: RS256

Refresh Token
* Type: Opaque UUID or cryptographically random string
* Lifetime: 30 days
* Storage: Database (hashed), client httpOnly cookie or secure storage
* Rotation: Issue new refresh token on each use, invalidate old token
* Metadata: user_id, device_info, ip_address, created_at, last_used_at, expires_at

Data Model
Users Table
- id (UUID, primary key)
- email (string, unique, indexed, nullable for mobile-only users)
- password_hash (string, nullable for mobile-only users)
- expo_device_id (string, unique, indexed, nullable for email/password users)
- created_at (timestamp)
- updated_at (timestamp)
- last_login_at (timestamp)

Refresh Tokens Table
- id (UUID, primary key)
- user_id (UUID, foreign key to users)
- token_hash (string, indexed)
- expires_at (timestamp, indexed)
- created_at (timestamp)
- last_used_at (timestamp)
- device_info (string)
- ip_address (string)

## Security Requirements
Password Security
* Passwords hashed using bcrypt (cost factor 12+)
* Minimum password length: 8 characters
* Password complexity requirements (configurable)

Token Security
* JWT signed with RS256 private key (minimum 2048-bit RSA key)
* Public key distributed to other services for stateless validation
* Refresh tokens hashed before database storage (SHA-256 or bcrypt)
* Refresh token reuse detection - revoke all user tokens on suspected compromise
* Token family tracking for multi-device support

Transport Security
* All endpoints require HTTPS in production
* Refresh tokens transmitted via httpOnly, secure cookies where possible
* Rate limiting on authentication endpoints to prevent brute force

## Mobile User Support
Device-Based Authentication
* Mobile users can register without providing email or password
* Authentication uses Expo Application Installation ID (from expo-application)
* Expo device ID stored securely and treated as a credential
* Device ID must be unique per installation
* Mobile-only users receive full access to service features

Account Upgrade Flow
* Mobile-only users can upgrade their account by adding email and password
* Upgrade endpoint: POST /users/{id}/upgrade with email and password in request body
* Email must be unique across all users (not already registered)
* Password must meet standard password requirements
* After upgrade, account becomes hybrid (all three fields populated)
* Once upgraded, users MUST authenticate using email/password (device ID authentication is disabled for hybrid accounts)
* This enforces secure authentication for multi-device access
* Existing refresh tokens remain valid after upgrade but new logins require email/password

Security Considerations
* Expo device ID treated as equivalent to password credential
* Device ID must be transmitted securely (HTTPS only)
* Loss of device ID (app reinstall) requires new account creation
* Consider device ID rotation/refresh mechanism for enhanced security
* Rate limiting applies to mobile authentication endpoints

Validation Rules
* Account type is determined by which fields are populated (no separate account_type field):
  - Mobile-only: expo_device_id is NOT NULL, email and password_hash are NULL
  - Email-only: email and password_hash are NOT NULL, expo_device_id is NULL
  - Hybrid: all three fields (email, password_hash, expo_device_id) are NOT NULL
* Database constraints enforce uniqueness of both email and expo_device_id
* Application logic derives authentication capabilities from field presence

Multi-Device Support
* Mobile-only (device ID) accounts are single-device only
* Users requiring multi-device access must upgrade to email/password authentication
* After upgrade to hybrid account, only email/password login is accepted (POST /auth/login)
* Device ID login (POST /auth/login/mobile) is rejected for hybrid accounts
* This ensures users have recoverable credentials when using multiple devices

## Cross-Service Integration
Token Validation for Other Services
* Other services (content, recommendations) validate JWTs independently using RS256 public key
* JWT public key distributed via HashiCorp Vault
* Private key remains secure within user service only (stored in Vault)
* No callback to auth service required for token validation
* Services extract user_id from validated token claims

Secrets Management
* HashiCorp Vault required for all production secrets
* JWT private/public key pairs stored in Vault
* Database credentials stored in Vault
* Services authenticate to Vault using appropriate auth method (AppRole, Kubernetes, etc.)
* Vault provides key rotation capabilities

Shared Authentication Package
* Lightweight Go package for JWT validation
* Middleware for protecting endpoints
* User context extraction utilities
* Distributed to content and recommendation services

Configuration
* JWT private/public key pair for RS256 signing (stored in HashiCorp Vault)
* Access token lifetime (environment variable, default 60 minutes)
* Refresh token lifetime (environment variable, default 30 days)
* Database connection parameters (stored in HashiCorp Vault)
* Password requirements (complexity, length)
* Rate limiting thresholds

## Non-Functional Requirements
Performance
* JWT validation must be stateless and not require database queries
* Token refresh operations should complete in <100ms
* Support for concurrent token validation across multiple service instances

Scalability
* Stateless design allows horizontal scaling
* Database queries only required for: login, registration, token refresh, and revocation
* Session-free architecture

Availability
* Service uptime target: 99.9%
* Graceful degradation if database temporarily unavailable (cached public keys for validation)

Monitoring
* Log all authentication attempts (success and failure)
* Track refresh token usage patterns
* Alert on suspicious activity (multiple failed logins, token reuse)

Future Considerations
* OAuth2/OpenID Connect support for third-party authentication
* Multi-factor authentication (MFA)
* Email verification workflow
* Password reset functionality
* Account lockout after failed attempts
* Remember device functionality​​​​​​​​​​​​​​​​






