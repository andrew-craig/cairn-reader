# Engineering Principles

**Version:** 1.0
**Last Updated:** 2026-10-15
**Target Audience:** All engineers contributing to the Cairn codebase

## Table of Contents

1. [Introduction](#introduction)
2. [Core Architectural Principles](#core-architectural-principles)
3. [Technology Stack](#technology-stack)
4. [Development Standards](#development-standards)
5. [Testing Philosophy](#testing-philosophy)
6. [Code Review Guidelines](#code-review-guidelines)
7. [Common Patterns](#common-patterns)
8. [Anti-Patterns to Avoid](#anti-patterns-to-avoid)

---

## Introduction

This document defines the engineering principles, architectural decisions, and coding standards for the Cairn project. These principles guide consistent decision-making and ensure code quality across our microservices architecture.

**Philosophy:** We prioritize simplicity, testability, and maintainability. Every pattern and decision documented here exists to solve real problems we've encountered, not to follow trends.

**How to Use This Document:**
- **Before starting a task:** Review relevant sections to understand established patterns
- **When making architectural decisions:** Consult rationale sections to understand "why"
- **During code review:** Reference specific sections when discussing code quality
- **When uncertain:** Follow the principle "consistency over perfection"

---

## Core Architectural Principles

### 1. Microservices with Database-Per-Service

**Decision:** Each service owns its own database. Services communicate only via HTTP APIs.

**Rationale:**
- **Independent scaling:** Services scale based on their own load
- **Clear ownership:** No ambiguity about data responsibility
- **Deployment independence:** Database schema changes don't cascade across services
- **Technology flexibility:** Each service can choose the optimal database technology

**Example:**
```
Read Service:
  Content Service (8080) → content_service
  Ingest RSS (8081) → rss_fetcher_service

User Service (8080) → User DB (postgres:5432)
```

**Implementation:**
- Ingest RSS manages RSS feeds and subscriptions in `rss_fetcher_service`
- Content Service stores articles, user content and Feed/Reads routes in `content_service`
- User Service manages authentication in `cairn_users`
- Services never directly query other services' databases
- Data sharing happens via REST APIs only

**Files:**
- `infrastructure/docker/dev/docker-compose.yml` (one Postgres instance, a logical database per service)
- `infrastructure/docker/scripts/init-databases.sh` (database initialization)

---

### 2. Layered Architecture (Handler → Service → Repository)

**Decision:** All Go services follow a strict three-layer architecture.

**Rationale:**
- **Testability:** Each layer can be tested independently
- **Single Responsibility:** HTTP concerns separated from business logic separated from data access
- **Flexibility:** Can swap implementations (e.g., change database) without touching business logic
- **Clarity:** New developers immediately understand code organization

**Layers:**

1. **HTTP Layer** (`internal/api/` or `internal/handlers/`)
   - HTTP request/response handling
   - Input validation
   - Error formatting
   - Authentication middleware

2. **Service Layer** (`internal/services/` or domain packages like `internal/processor/`)
   - Business logic
   - Orchestrates multiple repositories
   - Domain-specific error handling
   - No HTTP dependencies

3. **Repository Layer** (`internal/database/` or `internal/repository/`)
   - Database queries
   - Data mapping
   - Transaction management
   - No business logic

**Example:**
```go
// Handler (HTTP Layer)
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    var req LoginRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        api.WriteError(w, http.StatusBadRequest, api.ErrCodeBadRequest, "invalid request", nil, "v1")
        return
    }

    response, err := h.authService.Login(r.Context(), req.Email, req.Password, ...)
    if errors.Is(err, services.ErrInvalidCredentials) {
        api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "invalid credentials", nil, "v1")
        return
    }
    api.WriteSuccess(w, http.StatusOK, response, "v1")
}

// Service (Business Logic Layer)
func (s *AuthServiceImpl) Login(ctx context.Context, email, password string, ...) (*AuthResponse, error) {
    user, err := s.userRepo.GetByEmail(ctx, email)
    if err != nil {
        return nil, services.ErrInvalidCredentials
    }

    if !auth.VerifyPassword(user.PasswordHash, password) {
        return nil, services.ErrInvalidCredentials
    }

    tokens, err := s.jwtManager.GenerateTokenPair(user.ID)
    // ... business logic
    return &AuthResponse{User: user, AccessToken: tokens.AccessToken, ...}, nil
}

// Repository (Data Access Layer)
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
    query := `SELECT id, email, password_hash, created_at FROM users WHERE email = $1`

    var user models.User
    err := r.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, ...)
    if err == pgx.ErrNoRows {
        return nil, apperrors.ErrUserNotFound  // pkg/errors
    }
    return &user, nil
}
```

**Files:**
- `services/users/internal/handlers/auth_handler.go:48-80` (handler example)
- `services/users/internal/services/auth_service.go:75-120` (service example)
- `services/users/internal/database/user_repository.go:38-60` (repository example)

---

### 3. Dependency Injection

**Decision:** All dependencies are injected via constructors, not globals or singletons.

**Rationale:**
- **Testability:** Easy to mock dependencies in tests
- **Explicitness:** Dependencies are visible in function signatures
- **Flexibility:** Can swap implementations (e.g., test vs. production database)
- **No hidden coupling:** Impossible to use undeclared dependencies

**Example:**
```go
// main.go
db := setupDatabase()
userRepo := database.NewUserRepository(db)
authService := services.NewAuthService(services.AuthServiceConfig{
    UserRepo:            userRepo,
    RefreshTokenService: refreshTokenService,
    JWTManager:          jwtManager,
    PasswordHasher:      passwordHasher,
})
authHandler := handlers.NewAuthHandler(authService)

// Handler constructor
type AuthHandler struct {
    authService services.AuthService
}

func NewAuthHandler(authService services.AuthService) *AuthHandler {
    return &AuthHandler{authService: authService}
}

// Service constructor (takes a config struct of dependencies)
type AuthServiceConfig struct {
    UserRepo            database.UserRepository
    RefreshTokenService *auth.RefreshTokenService
    JWTManager          *auth.JWTManager
    PasswordHasher      *auth.PasswordHasher
    // ... tuning options (lockout threshold, password rules)
}

func NewAuthService(config AuthServiceConfig) AuthService {
    return &authService{
        userRepo:            config.UserRepo,
        refreshTokenService: config.RefreshTokenService,
        jwtManager:          config.JWTManager,
        passwordHasher:      config.PasswordHasher,
        // ...
    }
}
```

**Never do this:**
```go
// ❌ BAD: Global database instance
var DB *sql.DB

func GetUser(id string) (*User, error) {
    return DB.QueryRow("SELECT * FROM users WHERE id = $1", id)
}
```

**Files:**
- `services/users/cmd/user-service/main.go:80-140` (DI setup)
- `services/read/content/cmd/content/main.go` (DI setup)

---

### 4. Context-Driven Cancellation

**Decision:** All database operations and external API calls accept `context.Context` as the first parameter.

**Rationale:**
- **Request timeouts:** Prevents slow queries from blocking indefinitely
- **Graceful shutdown:** Can cancel in-flight operations during shutdown
- **Request tracing:** Can propagate request IDs and tracing information
- **Resource cleanup:** Database connections released when context cancelled

**Example:**
```go
// ✅ GOOD: All operations accept context
func (r *ArticleRepository) GetByID(ctx context.Context, id string) (*models.Article, error) {
    query := `SELECT id, title, link FROM articles WHERE id = $1`

    var article models.Article
    err := r.db.QueryRowContext(ctx, query, id).Scan(&article.ID, &article.Title, &article.Link)
    if err == sql.ErrNoRows {
        return nil, fmt.Errorf("article not found")
    }
    return &article, nil
}

// Handler extracts context from HTTP request
func (h *ArticleHandler) GetArticle(w http.ResponseWriter, r *http.Request) {
    articleID := chi.URLParam(r, "id")

    // Use request context (inherits timeout, cancellation)
    article, err := h.articleRepo.GetByID(r.Context(), articleID)
    if err != nil {
        api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "article not found", nil, "v1")
        return
    }
    api.WriteSuccess(w, http.StatusOK, article, "v1")
}
```

**Files:**
- All repository methods in `services/*/internal/database/*_repository.go` and `services/read/content/internal/repository/*.go`
- All service methods in `services/*/internal/services/*.go`

---

### 5. Error Handling: Wrap and Return, Handle at Top

**Decision:** Errors flow up the call stack with context. Only handlers convert errors to HTTP responses.

**Rationale:**
- **Context preservation:** Error wrapping preserves the full error chain
- **Separation of concerns:** Business logic doesn't know about HTTP status codes
- **Reusability:** Same service method can be used by HTTP, gRPC, CLI, etc.
- **Consistent error responses:** Centralized error formatting in handlers

**Pattern:**

1. **Repository layer:** Wrap errors with context
2. **Service layer:** Define domain errors, wrap repository errors
3. **Handler layer:** Convert domain errors to HTTP responses

**Example:**
```go
// Repository: Wrap with context
func (r *FeedRepository) GetNextFeed(ctx context.Context) (*models.Feed, error) {
    // ... query
    if err != nil {
        return nil, fmt.Errorf("failed to get next feed: %w", err)
    }
    return feed, nil
}

// Service: Define domain errors
var (
    ErrInvalidCredentials = errors.New("invalid credentials")
    ErrAccountExists = errors.New("account already exists")
)

func (s *AuthService) Login(...) (*AuthResponse, error) {
    user, err := s.userRepo.GetByEmail(ctx, email)
    if err != nil {
        return nil, ErrInvalidCredentials  // Don't leak "user not found"
    }

    if !auth.VerifyPassword(user.PasswordHash, password) {
        return nil, ErrInvalidCredentials
    }
    return response, nil
}

// Handler: Convert to HTTP status codes
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
    response, err := h.authService.Login(r.Context(), ...)
    if errors.Is(err, services.ErrInvalidCredentials) {
        api.WriteError(w, http.StatusUnauthorized, api.ErrCodeUnauthorized, "invalid credentials", nil, "v1")
        return
    }
    if errors.Is(err, services.ErrAccountExists) {
        api.WriteError(w, http.StatusConflict, api.ErrCodeConflict, "account already exists", nil, "v1")
        return
    }
    if err != nil {
        logging.FromContext(r.Context()).Error("login failed", slog.Any("error", err))
        api.WriteError(w, http.StatusInternalServerError, api.ErrCodeInternal, "internal error", nil, "v1")
        return
    }
    api.WriteSuccess(w, http.StatusOK, response, "v1")
}

// In the User Service this chain of errors.Is checks lives in one place:
// the ordered serviceErrorTable in handlers/errors.go, walked by writeServiceError.
```

**Never do this:**
```go
// ❌ BAD: Repository knows about HTTP status codes
func (r *UserRepository) GetByEmail(email string) (*User, int, error) {
    // ...
    if err == sql.ErrNoRows {
        return nil, http.StatusNotFound, errors.New("user not found")
    }
}
```

**Files:**
- `services/users/internal/services/auth_service.go:15-20` (domain errors)
- `services/users/internal/handlers/auth_handler.go` (error conversion)
- `services/read/content/internal/repository/user_content.go` (error wrapping)

---

### 6. Stateless JWT Authentication

**Decision:** Use JWT tokens with RS256 signing for authentication. Store refresh tokens in database.

**Rationale:**
- **Scalability:** No session storage required, services can scale horizontally
- **Security:** RS256 uses public/private key pairs (private key never leaves user service)
- **Verification:** Other services verify tokens with public key (no network call)
- **Revocation:** Refresh tokens enable logout without blacklisting JWTs

**Architecture:**
```
User Service:
  - Signs JWTs with private key (from Vault)
  - Stores refresh tokens (hashed) in database
  - Validates refresh tokens on /auth/refresh

Other Services:
  - Verify JWTs with public key (from Vault)
  - Extract user_id from claims
  - No database lookup needed for authentication
```

**Example:**
```go
// Generate token pair (User Service)
func (j *JWTManager) GenerateTokenPair(userID string) (*TokenPair, error) {
    // Access token: short-lived (60 min default), contains user_id claim
    accessToken, err := j.GenerateAccessToken(userID)

    // Refresh token: long-lived (30 days default), random string hashed before storage
    refreshToken, err := generateSecureToken()

    return &TokenPair{
        AccessToken:  accessToken,
        RefreshToken: refreshToken,
    }, nil
}

// Verify token (Any Service)
func (v *Validator) ValidateToken(tokenString string) (*Claims, error) {
    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
            return nil, fmt.Errorf("unexpected signing method")
        }
        return v.publicKey, nil
    })

    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return &Claims{UserID: claims["user_id"].(string)}, nil
    }
    return nil, errors.New("invalid token")
}
```

**Token Lifetime:**
- Access Token: 60 minutes by default (`JWT_ACCESS_TOKEN_EXPIRY`; short-lived, cannot be revoked)
- Refresh Token: 30 days by default (`JWT_REFRESH_TOKEN_EXPIRY`; stored in DB, can be revoked)

**Files:**
- `services/users/internal/auth/jwt.go` (JWT generation)
- `pkg/auth/middleware.go` (JWT validation)
- `infrastructure/docker/scripts/init-vault.sh` (RSA key generation)

---

## Technology Stack

### Backend Services (Go)

**Go Version:** 1.25 (see each module's `go.mod`; some `pkg/` modules still declare 1.24)

**Core Libraries:**

| Library | Version | Purpose | Rationale |
|---------|---------|---------|-----------|
| `database/sql` | stdlib | Database abstraction | Standard library, no external deps |
| `github.com/lib/pq` | v1.10.9 | PostgreSQL driver | Most mature Go postgres driver |
| `github.com/jackc/pgx/v5` | v5.10.0 | PostgreSQL (User Service) | Better performance, native types |
| `github.com/go-chi/chi/v5` | v5.2.3 | HTTP router (all services) | Lightweight, idiomatic, stdlib-compatible |
| `log/slog` | stdlib | Structured logging | Standard library, structured, zero-config |
| `github.com/golang-jwt/jwt/v5` | v5.3.0 | JWT tokens | Industry standard |
| `github.com/mmcdole/gofeed` | v1.3.0 | RSS parsing | Handles multiple feed formats |
| `github.com/golang-migrate/migrate/v4` | v4.19.1 | Database migrations | Production-grade migration tool |
| `github.com/hashicorp/vault/api` | v1.22.0 | Secrets management | Industry standard secrets vault |

**Testing:**
- `testing` (stdlib) - All unit tests
- `net/http/httptest` (stdlib) - HTTP handler testing
- `github.com/stretchr/testify` - Assertions and mocking

**Read Service Specific Libraries:**

| Library | Version | Purpose | Rationale |
|---------|---------|---------|-----------|
| `github.com/go-chi/chi/v5` | v5.2.3 | HTTP router | Lightweight, idiomatic Go, excellent middleware support, stdlib-compatible |
| `github.com/go-shiori/go-readability` | latest | Content extraction | Extract readable content from HTML articles |
| `github.com/microcosm-cc/bluemonday` | v1.0.27 | HTML sanitization | Security - XSS prevention, industry standard |
| `github.com/sony/gobreaker` | v0.5.0 | Circuit breaker | Resilience for external API calls, prevents cascade failures |
| `github.com/robfig/cron/v3` | v3.0.1 | Job scheduling | Background jobs (cleanup, polling), cron-compatible syntax |

**Why chi/v5?**
- Minimal overhead compared to full frameworks
- 100% compatible with stdlib http.Handler
- Composable middleware chain
- Path parameter extraction without string manipulation
- Suitable for services with moderate routing needs; used by the User, Content, Fetcher and Email services and the self-host binary

**Why Go?**
- **Performance:** Fast startup, low memory footprint
- **Concurrency:** Native goroutines for background tasks (feed fetching)
- **Deployment:** Single binary with no runtime dependencies
- **Ecosystem:** Excellent database, HTTP, and testing libraries

---

### Mobile App (React Native)

**React Native Version:** 0.81.5
**Expo SDK:** ~54.0.32
**React Version:** 19.1.0
**TypeScript Version:** ^5.3.0

**Core Libraries:**

| Library | Version | Purpose | Rationale |
|---------|---------|---------|-----------|
| `@react-navigation/native` | ^6.1.9 | Navigation | Industry standard, type-safe |
| `@react-navigation/stack` | ^6.3.20 | Stack navigation | Standard screen transitions |
| `@react-navigation/bottom-tabs` | ^6.5.11 | Tab navigation | Common mobile pattern |
| `@react-native-async-storage/async-storage` | 2.2.0 | Local storage | Persistent key-value storage |
| `expo-application` | ~6.0.7 | Device info | Device ID for authentication |
| `@expo/vector-icons` | ^15.0.3 | Icons | Comprehensive icon set |
| `@expo-google-fonts/inter` | ^0.4.2 | Typography | Modern, readable font |

**Development Tools:**
- `@typescript-eslint/eslint-plugin` - TypeScript linting
- `eslint` - Code quality
- `typescript` - Type checking

**Why React Native + Expo?**
- **Cross-platform:** Single codebase for iOS and Android
- **Fast iteration:** Hot reload, over-the-air updates
- **Type safety:** TypeScript throughout
- **Developer experience:** Expo simplifies native module integration
- **Community:** Large ecosystem of libraries and components

---

### Database (PostgreSQL)

**PostgreSQL Version:** 16-alpine

**Why PostgreSQL?**
- **Reliability:** ACID compliance, proven at scale
- **Features:** JSONB, full-text search, arrays (used for categories)
- **Tooling:** Excellent Go drivers, migration tools
- **Operations:** Mature backup, replication, monitoring tools

**Database Design Patterns:**
- Separate databases per service (see Architecture Principles)
- Migrations managed via golang-migrate (Users, Read)
- Connection pooling configured per service
- Indexes on commonly queried columns

**Files:**
- `services/read/content/migrations/` (content schema)
- `services/read/fetcher/migrations/` (RSS ingest schema)
- `services/users/migrations/` (user schema)

---

### Infrastructure

**Docker:** Latest stable
**Docker Compose:** v2.x
**HashiCorp Vault:** 1.18 (dev mode in development, production setup required)

**Why Docker?**
- **Consistency:** Same environment across dev, staging, production
- **Isolation:** Each service in its own container
- **Orchestration:** Docker Compose simplifies multi-service development

**Why Vault?**
- **Secrets management:** JWT keys, database passwords centralized
- **Rotation:** Built-in secret rotation capabilities
- **Audit:** All secret access is logged

**Files:**
- `infrastructure/docker/dev/docker-compose.yml` (multi-service orchestration)

---

## Development Standards

### Code Organization

#### Go Services

**Directory Structure:**
```
services/{service}/
├── cmd/{service}/main.go        # Application entry point
├── cmd/migrate/main.go          # Migration tool (User Service only)
├── internal/                    # Private application code
│   ├── api/                     # HTTP handlers, middleware (Read Service)
│   ├── handlers/                # HTTP handlers (User Service)
│   ├── services/                # Business logic (User Service)
│   ├── db/ or database/         # Repository layer
│   ├── {domain}/                # Domain-specific logic (fetcher/, processor/)
│   ├── config/                  # Configuration (User Service)
│   └── testutil/                # Test utilities
├── migrations/                  # Database migrations
├── Dockerfile                   # Multi-stage build
├── Makefile                     # Build automation
├── go.mod, go.sum              # Dependencies
└── README.md                    # Service documentation
```

**Package Rules:**
1. **`internal/`** - Service-specific code, cannot be imported by other services
2. **`pkg/`** (repo root, one Go module per package: `api`, `auth`, `config`, `env`, `errors`, `logging`, `middleware`, `models`, `rss`) - Reusable code shared across services
3. **`cmd/`** - Application entry points (main.go files)

**Import Order:**
```go
import (
    // 1. Standard library
    "context"
    "database/sql"
    "fmt"

    // 2. Third-party libraries
    "github.com/go-chi/chi/v5"
    "github.com/lib/pq"

    // 3. Internal packages
    "github.com/andrew-craig/cairn-reader/services/users/internal/database"
    "github.com/andrew-craig/cairn-reader/services/users/internal/services"
)
```

---

#### Mobile App

**Directory Structure:**
```
apps/mobile/
├── src/
│   ├── components/              # Reusable UI components
│   │   ├── common/              # Generic components (Button, Card, etc.)
│   │   └── {feature}/           # Feature-specific components
│   ├── screens/                 # Full-screen components
│   ├── navigation/              # Navigation configuration
│   │   ├── RootNavigator.tsx   # Stack navigator
│   │   └── TabNavigator.tsx    # Bottom tabs
│   ├── services/                # API clients and storage
│   ├── contexts/                # React contexts (auth, theme)
│   ├── hooks/                   # Custom hooks
│   ├── types/                   # TypeScript type definitions
│   ├── constants/               # App constants (theme, config)
│   ├── utils/                   # Utility functions
│   └── config/                  # Configuration (API URLs)
├── assets/                      # Images, fonts, static assets
├── App.tsx                      # Application entry point
├── app.json                     # Expo configuration
├── package.json                 # Dependencies
└── tsconfig.json                # TypeScript config
```

**File Naming:**
- **Components:** `PascalCase.tsx` (e.g., `Button.tsx`, `ArticleCard.tsx`)
- **Screens:** `PascalCaseScreen.tsx` (e.g., `LoginScreen.tsx`)
- **Services:** `camelCase.ts` (e.g., `auth.ts`, `storage.ts`)
- **Types:** `camelCase.ts` (e.g., `article.ts`, `navigation.ts`)

---

### Naming Conventions

#### Go

**Files:**
- Lowercase with underscores: `user_repository.go`, `auth_service.go`
- Test files: `{source}_test.go` (e.g., `user_repository_test.go`)
- Integration tests: `{source}_integration_test.go` or standalone `integration_test.go`

**Variables and Functions:**
- **Exported (public):** `PascalCase` - `NewUserRepository`, `GetByEmail`
- **Unexported (private):** `camelCase` - `validatePassword`, `hashToken`
- **Constants:** `PascalCase` or `ALL_CAPS` - `DefaultPort` or `MAX_RETRIES`
- **Acronyms:** Follow case (HTTP → `ServeHTTP`, `httpClient`, not `HTTPClient`)

**Interfaces:**
- Prefer noun names: `UserRepository`, `AuthService`
- Single-method interfaces often end in `-er`: `Fetcher`, `Parser`

**Example:**
```go
// ✅ GOOD
type UserRepository interface {
    GetByEmail(ctx context.Context, email string) (*models.User, error)
    Create(ctx context.Context, user *models.User) error
}

type userRepositoryImpl struct {
    pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
    return &userRepositoryImpl{pool: pool}
}

const MaxLoginAttempts = 5
```

---

#### TypeScript / React Native

**Interfaces and Types:**
- `PascalCase` for interfaces and types
- Use `interface` for objects, `type` for unions/primitives

```typescript
// ✅ GOOD
export interface Article {
  id: string;
  title: string;
  isRead: boolean;
}

export type SortOption = 'recent' | 'oldest' | 'title';
export type FilterOption = 'all' | 'unread' | 'read';
```

**Components:**
- `PascalCase` for component names and files
- Props interfaces named `{Component}Props`

```typescript
// ✅ GOOD: Button.tsx
interface ButtonProps {
  title: string;
  onPress: () => void;
  variant?: 'primary' | 'secondary';
}

export const Button: React.FC<ButtonProps> = ({ title, onPress, variant = 'primary' }) => {
  // ...
};
```

**Functions and Variables:**
- `camelCase` for functions and variables
- Event handlers prefixed with `handle`: `handlePress`, `handleSubmit`
- Boolean variables prefixed with `is`, `has`, `should`: `isLoading`, `hasError`

```typescript
// ✅ GOOD
const [isLoading, setIsLoading] = useState(false);
const [hasError, setHasError] = useState(false);

const handleSubmit = async () => {
  setIsLoading(true);
  try {
    await AuthService.login(email, password);
  } catch (error) {
    setHasError(true);
  } finally {
    setIsLoading(false);
  }
};
```

---

### Code Style

#### Go

**Formatting:**
- **Always run `go fmt` before committing** (enforced by `make lint`)
- Use `gofmt -s` for simplification
- Line length: No hard limit, but prefer readability over brevity

**Comments:**
- All exported functions/types must have doc comments
- Doc comments start with the name of the thing being described
- Use `//` for single-line, `/* */` for multi-line rarely

```go
// ✅ GOOD
// GetByEmail retrieves a user by their email address.
// Returns ErrUserNotFound if no user exists with the given email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
    // ...
}
```

**Error Messages:**
- Lowercase, no ending punctuation
- Provide context, not just "failed"

```go
// ✅ GOOD
return fmt.Errorf("failed to fetch feed %s: %w", feedURL, err)

// ❌ BAD
return fmt.Errorf("Error: failed")
```

**Struct Initialization:**
- Use field names for clarity (except very small structs)

```go
// ✅ GOOD
user := &models.User{
    ID:        uuid.New().String(),
    Email:     email,
    CreatedAt: time.Now(),
}

// ❌ BAD (unclear what each value represents)
user := &models.User{uuid.New().String(), email, time.Now(), time.Now()}
```

---

#### TypeScript / React Native

**TypeScript Configuration:**
- **Strict mode enabled** (`strict: true` in tsconfig.json)
- All props and state must be typed (no `any`)
- Use `interface` for object shapes, `type` for unions

**Component Style:**
- **Always use functional components** (no class components)
- Use hooks for state and lifecycle
- Destructure props in function parameters

```typescript
// ✅ GOOD
interface ArticleCardProps {
  article: Article;
  onPress: () => void;
}

export const ArticleCard: React.FC<ArticleCardProps> = ({ article, onPress }) => {
  const colorScheme = useColorScheme();
  const colors = colorScheme === 'dark' ? Colors.dark : Colors.light;

  return (
    <TouchableOpacity onPress={onPress}>
      <Text>{article.title}</Text>
    </TouchableOpacity>
  );
};

// ❌ BAD
export class ArticleCard extends React.Component {
  // Don't use class components
}
```

**State Management:**
- Use `useState` for local state
- Use Context API for global state (auth, theme)
- Don't use Redux unless truly necessary (not currently in the stack)

```typescript
// ✅ GOOD: Local state
const [loading, setLoading] = useState(false);
const [articles, setArticles] = useState<Article[]>([]);

// ✅ GOOD: Global state
const { user, isAuthenticated, login } = useAuth();
```

**Async Operations:**
- Use `async/await` (not `.then()`)
- Always handle errors with try/catch
- Show loading states and error messages

```typescript
// ✅ GOOD
const handleLogin = async () => {
  setIsLoading(true);
  setError(null);

  try {
    await AuthService.loginWithDevice();
    navigation.navigate('MainTabs');
  } catch (error) {
    setError(error instanceof Error ? error.message : 'Login failed');
  } finally {
    setIsLoading(false);
  }
};

// ❌ BAD
const handleLogin = () => {
  AuthService.loginWithDevice()
    .then(() => navigation.navigate('MainTabs'))
    .catch(err => console.log(err));  // No loading state, no error handling
};
```

---

### Configuration Management

#### Environment Variables

**Go Services:**
- Use the `pkg/env` helpers (`env.GetString`, `GetInt`, `GetBool`, `GetDuration`) with sensible defaults
- Group related config into structs (User Service pattern)
- Validate configuration on startup

```go
// ✅ GOOD: Centralized config (User Service pattern)
type Config struct {
    Server   ServerConfig
    Database DatabaseConfig
    JWT      JWTConfig
}

func Load() (*Config, error) {
    cfg := &Config{
        Server: ServerConfig{
            Port: env.GetString("PORT", "8080"),
        },
        Database: DatabaseConfig{
            Host: env.GetString("DB_HOST", "localhost"),
        },
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid configuration: %w", err)
    }

    return cfg, nil
}
```

**Mobile App:**
- Use `src/config/` for startup wiring; the server URL state lives in `@cairn/shared`
- Use Expo constants for environment-specific config
- Never commit secrets to version control

```typescript
// ✅ GOOD: apps/mobile/src/config/storage.ts
// The backend the mobile app ships against by default.
export const DEFAULT_SERVER_URL = 'https://cairn.seatrain.net';
```

**Secrets Management:**
- **Development:** Use `.env` files (never commit to git)
- **Production:** Use HashiCorp Vault
- **Mobile:** Use Expo SecureStore for sensitive data

**Files:**
- `services/users/internal/config/config.go` (structured config example)
- `infrastructure/docker/dev/docker-compose.yml` (environment variable definitions)

---

## Testing Philosophy

### Overall Philosophy

**Core Principle:** Write tests for all business logic before code review. Tests are not optional.

**Why We Test:**
1. **Confidence:** Refactor without fear of breaking things
2. **Documentation:** Tests show how code is intended to be used
3. **Design feedback:** Hard-to-test code is often poorly designed
4. **Regression prevention:** Bugs fixed should have tests preventing recurrence

**Coverage Expectations:**
- **Business logic (services):** 80%+ coverage required
- **Repositories:** Integration tests with real database preferred
- **Handlers:** Test error cases and validation, not just happy path
- **Utilities:** 100% coverage (they're reused everywhere)

---

### Go Testing Standards

#### Test Organization

**File Structure:**
```
internal/
├── services/
│   ├── auth_service.go
│   └── auth_service_test.go       # Unit tests (mocked dependencies)
├── database/
│   ├── user_repository.go
│   ├── user_repository_test.go    # Unit tests
│   └── user_repository_integration_test.go  # Integration tests (real DB)
└── handlers/
    ├── auth_handler.go
    └── auth_handler_test.go       # HTTP handler tests
```

**Naming:**
- Test files: `{source}_test.go`
- Integration tests: `{source}_integration_test.go` or standalone `integration_test.go`
- Test functions: `Test{FunctionName}` or `Test{FunctionName}_{Scenario}`

**Example:**
```go
func TestLogin_Success(t *testing.T) { ... }
func TestLogin_InvalidCredentials(t *testing.T) { ... }
func TestLogin_AccountLocked(t *testing.T) { ... }
```

---

#### Test Patterns

**1. Table-Driven Tests**

Use for testing multiple scenarios of the same function:

```go
func TestValidatePassword(t *testing.T) {
    tests := []struct {
        name     string
        password string
        wantErr  bool
    }{
        {
            name:     "valid password",
            password: "SecureP@ssw0rd",
            wantErr:  false,
        },
        {
            name:     "too short",
            password: "Short1!",
            wantErr:  true,
        },
        {
            name:     "no special character",
            password: "NoSpecialChar1",
            wantErr:  true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := auth.ValidatePassword(tt.password)
            if (err != nil) != tt.wantErr {
                t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

**Files:** `services/users/internal/auth/password_test.go:15-50`

---

**2. Repository Integration Tests**

Test repositories with a real database, not mocks:

```go
func TestUserRepository_Create(t *testing.T) {
    db := testutil.SetupTestDB(t)
    defer db.Close()
    defer testutil.CleanupTestDB(t, db)

    repo := database.NewUserRepository(db)

    user := &models.User{
        ID:    uuid.New().String(),
        Email: "test@example.com",
    }

    err := repo.Create(context.Background(), user)
    if err != nil {
        t.Fatalf("Create() failed: %v", err)
    }

    // Verify user was created
    fetched, err := repo.GetByEmail(context.Background(), user.Email)
    if err != nil {
        t.Fatalf("GetByEmail() failed: %v", err)
    }

    if fetched.ID != user.ID {
        t.Errorf("got ID %s, want %s", fetched.ID, user.ID)
    }
}
```

**Why real databases?**
- Catches SQL errors that mocks can't
- Tests actual query performance
- Validates constraints and indexes

**Files:** `services/read/content/internal/repository/routing_integration_test.go`

---

**3. Service Unit Tests (Mocked Dependencies)**

Test business logic in isolation:

```go
func TestAuthService_Login_InvalidPassword(t *testing.T) {
    // Setup mocks
    mockUserRepo := new(MockUserRepository)
    mockJWT := new(MockJWTManager)

    service := services.NewAuthService(mockUserRepo, mockJWT, nil)

    // Configure mock behavior
    user := &models.User{
        ID:           "user-123",
        Email:        "test@example.com",
        PasswordHash: "$2a$12$hashedpassword",
    }
    mockUserRepo.On("GetByEmail", mock.Anything, "test@example.com").Return(user, nil)

    // Test
    _, err := service.Login(context.Background(), "test@example.com", "wrongpassword", "", "")

    // Verify
    if !errors.Is(err, services.ErrInvalidCredentials) {
        t.Errorf("expected ErrInvalidCredentials, got %v", err)
    }

    mockUserRepo.AssertExpectations(t)
}
```

**Files:** `services/users/internal/services/auth_service_test.go`

---

**4. HTTP Handler Tests**

Use `httptest` to test HTTP handlers:

```go
func TestHandleHealth(t *testing.T) {
    req := httptest.NewRequest("GET", "/health", nil)
    w := httptest.NewRecorder()

    server := &Server{}
    server.handleHealth(w, req)

    resp := w.Result()
    if resp.StatusCode != http.StatusOK {
        t.Errorf("expected status 200, got %d", resp.StatusCode)
    }

    var response map[string]string
    json.NewDecoder(resp.Body).Decode(&response)

    if response["status"] != "healthy" {
        t.Errorf("expected status healthy, got %s", response["status"])
    }
}
```

---

#### Test Utilities

**Shared Test Helpers:**

Create `internal/testutil/helpers.go`:

```go
package testutil

import (
    "database/sql"
    "testing"
)

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *sql.DB {
    t.Helper()

    connStr := "host=localhost port=5432 user=test password=test dbname=test_db sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        t.Fatalf("failed to connect to test database: %v", err)
    }

    if err := db.Ping(); err != nil {
        t.Fatalf("failed to ping test database: %v", err)
    }

    return db
}

// CleanupTestDB removes all test data
func CleanupTestDB(t *testing.T, db *sql.DB) {
    t.Helper()

    queries := []string{
        "DELETE FROM user_articles",  // Dependent tables first
        "DELETE FROM articles",
        "DELETE FROM users",
    }

    for _, query := range queries {
        if _, err := db.Exec(query); err != nil {
            t.Logf("cleanup failed: %v", err)
        }
    }
}

// Mark functions with t.Helper() to exclude from error line numbers
```

**Files:** `services/read/content/internal/testutil/database.go`

---

#### Running Tests

**Commands:**
```bash
# Run all tests
make test
go test ./...

# Run with coverage
make test-coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package
go test -v ./internal/services

# Run specific test
go test -v ./internal/services -run TestLogin_Success

# Integration tests only
make test-integration
go test -v ./... -run Integration
```

**CI Requirements (when implemented):**
- All tests must pass before merge
- Coverage must not decrease
- No skipped tests without explanation

---

### Mobile Testing Standards

**Current State:** Jest (`jest-expo`) and `@testing-library/react-native` are installed; run `npm test` (or `npm run test:ci` for coverage). Test files sit next to their source (`Button.test.tsx`).

**Expectations:**

1. **Component tests:** Test all components in `src/components/common/`
2. **Screen tests:** Test user interactions and state changes
3. **Service tests:** Test API clients and storage services
4. **Hook tests:** Test custom hooks (useAuth, etc.)

**Example:**
```typescript
// Button.test.tsx
import { render, fireEvent } from '@testing-library/react-native';
import { Button } from './Button';

describe('Button', () => {
  it('calls onPress when pressed', () => {
    const onPress = jest.fn();
    const { getByText } = render(<Button title="Click me" onPress={onPress} />);

    fireEvent.press(getByText('Click me'));

    expect(onPress).toHaveBeenCalledTimes(1);
  });

  it('shows loading indicator when loading', () => {
    const { getByTestId } = render(<Button title="Click" onPress={() => {}} loading />);

    expect(getByTestId('loading-indicator')).toBeTruthy();
  });
});
```

**Coverage Expectations:**
- Components: 80%+ coverage
- Services: 90%+ coverage (API clients, storage)
- Utilities: 100% coverage

---

## Code Review Guidelines

### Philosophy

**Goal:** Maintain code quality while enabling rapid iteration

**Tone:** Constructive, educational, collaborative (never confrontational)

**Responsibilities:**
- **Author:** Write clear, tested code with good commit messages
- **Reviewer:** Provide actionable feedback, ask questions, approve promptly

---

### Pre-Review Checklist (Author)

Before requesting review, ensure:

**✅ Code Quality:**
- [ ] All tests pass (`make test` for Go, `npm run type-check` for mobile)
- [ ] Code is formatted (`make fmt` for Go, `npm run lint` for mobile)
- [ ] No commented-out code or debug statements
- [ ] No hardcoded credentials or secrets

**✅ Testing:**
- [ ] New features have tests (unit + integration where applicable)
- [ ] Bug fixes include regression tests
- [ ] Tests cover edge cases, not just happy path

**✅ Documentation:**
- [ ] Public functions have doc comments
- [ ] Complex logic has explanatory comments
- [ ] README updated if behavior changed
- [ ] API changes documented

**✅ Commits:**
- [ ] Commits are logical units (not "fix", "wip", "oops")
- [ ] Commit messages explain "why", not "what"
- [ ] No merge commits (rebase before pushing)

**Example Good Commit:**
```
Add JWT refresh token rotation

Implements automatic rotation of refresh tokens on each use to
improve security. Old tokens are invalidated immediately after
successful refresh.

Closes #42
```

---

### Review Checklist (Reviewer)

**🔍 Correctness:**
- [ ] Code does what the description says
- [ ] Edge cases are handled (nil, empty, error cases)
- [ ] No race conditions or concurrency issues
- [ ] Database queries are efficient (no N+1, proper indexes)

**🏗️ Design:**
- [ ] Follows established patterns (layered architecture, DI, error handling)
- [ ] Appropriate abstractions (not over-engineered)
- [ ] Single Responsibility Principle followed
- [ ] Code is in the right place (handler vs service vs repository)

**🔒 Security:**
- [ ] User input is validated
- [ ] SQL injection prevented (parameterized queries)
- [ ] Authentication/authorization enforced
- [ ] Secrets not hardcoded or logged
- [ ] Error messages don't leak sensitive info

**🧪 Testing:**
- [ ] Tests exist and are meaningful (not just 100% coverage)
- [ ] Tests use appropriate patterns (table-driven, mocking, etc.)
- [ ] Integration tests use real database where applicable

**📚 Readability:**
- [ ] Code is self-explanatory (good naming)
- [ ] Complex logic has comments explaining "why"
- [ ] No magic numbers (use named constants)
- [ ] Functions are short and focused

---

### Giving Feedback

**Levels of Feedback:**

1. **🚨 Blocker:** Must be fixed before merge
   - "This SQL query is vulnerable to injection attacks"
   - "This will cause a panic if user is nil"

2. **💡 Suggestion:** Should be fixed, but not critical
   - "Consider extracting this into a helper function for reusability"
   - "This could be simplified with a map instead of multiple if statements"

3. **🤔 Question:** Seeking clarification or discussion
   - "Why did we choose this approach over X?"
   - "Could this be simplified?"

4. **📝 Nit:** Style/preference, optional
   - "Typo: 'recieve' → 'receive'"
   - "This variable name could be more descriptive"

**Examples:**

```
# ✅ GOOD: Specific, actionable, kind
💡 Suggestion: Consider using `errors.Is()` here instead of string comparison.
Error messages can change, but error types are more stable.

# ❌ BAD: Vague, confrontational
This error handling is wrong.
```

```
# ✅ GOOD: Educational
🚨 Blocker: This query is vulnerable to SQL injection. Please use
parameterized queries like `db.Query("SELECT * FROM users WHERE id = $1", userID)`.
See services/users/internal/database/user_repository.go:45 for an example.

# ❌ BAD: No explanation
Fix this SQL injection vulnerability.
```

---

### Responding to Feedback

**As an Author:**

1. **Assume good intent:** Reviewers are trying to help, not criticize
2. **Ask for clarification:** If feedback is unclear, ask questions
3. **Push back respectfully:** If you disagree, explain your reasoning
4. **Update the PR:** Address feedback, don't just comment "fixed"
5. **Resolve conversations:** Mark conversations as resolved after addressing

**Example Response:**
```
> 💡 Consider caching this database query result

Good idea! I've added a 5-minute in-memory cache using sync.Map.
See the new `getCached()` method at line 120.
```

---

### Approval Criteria

**Approve if:**
- Code works correctly and follows patterns
- Tests exist and pass
- No security issues
- Minor issues (nits) don't block approval

**Request changes if:**
- Security vulnerabilities
- No tests for new functionality
- Breaks established patterns without justification
- Logic errors or bugs

**Comment only if:**
- Learning opportunity but code is acceptable
- Suggesting future improvements (not blocking)

---

### PR Size Guidelines

**Ideal PR Size:** 200-400 lines changed

**Why?**
- Easier to review thoroughly
- Faster to merge
- Lower risk of bugs

**Large PRs (>500 lines):**
- Break into multiple PRs if possible
- Provide detailed description and testing instructions
- Consider pairing with reviewer for walkthrough

**Very Large PRs (>1000 lines):**
- Should be rare (refactorings, generated code, etc.)
- Must have excellent test coverage
- Consider multiple reviewers

---

## Common Patterns

### Repository Pattern

**When:** All database access

**Implementation:**
```go
type UserRepository interface {
    Create(ctx context.Context, user *models.User) error
    GetByID(ctx context.Context, id string) (*models.User, error)
    GetByEmail(ctx context.Context, email string) (*models.User, error)
    Update(ctx context.Context, user *models.User) error
}

type userRepositoryImpl struct {
    pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) UserRepository {
    return &userRepositoryImpl{pool: pool}
}

func (r *userRepositoryImpl) GetByEmail(ctx context.Context, email string) (*models.User, error) {
    query := `SELECT id, email, password_hash FROM users WHERE email = $1`

    var user models.User
    err := r.pool.QueryRow(ctx, query, email).Scan(&user.ID, &user.Email, &user.PasswordHash)
    if errors.Is(err, pgx.ErrNoRows) {
        return nil, apperrors.ErrUserNotFound
    }
    if err != nil {
        return nil, fmt.Errorf("query user by email: %w", err)
    }

    return &user, nil
}
```

**Key Points:**
- Accept `context.Context` as first parameter
- Return domain errors (not database errors)
- Use parameterized queries ($1, $2, etc.)

**Files:** All `*_repository.go` files

---

### Service Pattern

**When:** Orchestrating business logic across multiple repositories

**Implementation:**
```go
type AuthService interface {
    Login(ctx context.Context, email, password string) (*AuthResponse, error)
    Register(ctx context.Context, email, password string) (*AuthResponse, error)
}

type authServiceImpl struct {
    userRepo         database.UserRepository
    refreshTokenRepo database.RefreshTokenRepository
    jwtManager       *auth.JWTManager
}

// Simplified; the real NewAuthService takes an AuthServiceConfig struct.
func NewAuthService(
    userRepo database.UserRepository,
    refreshTokenRepo database.RefreshTokenRepository,
    jwtManager *auth.JWTManager,
) AuthService {
    return &authServiceImpl{
        userRepo:         userRepo,
        refreshTokenRepo: refreshTokenRepo,
        jwtManager:       jwtManager,
    }
}

func (s *authServiceImpl) Login(ctx context.Context, email, password string) (*AuthResponse, error) {
    // 1. Get user
    user, err := s.userRepo.GetByEmail(ctx, email)
    if err != nil {
        return nil, services.ErrInvalidCredentials  // Don't leak "user not found"
    }

    // 2. Verify password
    if !auth.VerifyPassword(user.PasswordHash, password) {
        return nil, services.ErrInvalidCredentials
    }

    // 3. Generate tokens
    tokens, err := s.jwtManager.GenerateTokenPair(user.ID)
    if err != nil {
        return nil, fmt.Errorf("generate tokens: %w", err)
    }

    // 4. Store refresh token
    err = s.refreshTokenRepo.Create(ctx, tokens.RefreshToken, user.ID)
    if err != nil {
        return nil, fmt.Errorf("store refresh token: %w", err)
    }

    return &AuthResponse{User: user, AccessToken: tokens.AccessToken, RefreshToken: tokens.RefreshToken}, nil
}
```

**Key Points:**
- Business logic only (no HTTP concepts)
- Define domain errors
- Orchestrate multiple repositories
- Return domain models

**Files:** `services/users/internal/services/*.go`

---

### Middleware Pattern (Go)

**When:** Cross-cutting concerns (auth, logging, CORS)

**Implementation:**
```go
// Middleware signature
type Middleware func(http.Handler) http.Handler

// Logging middleware
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        requestID := uuid.New().String()
        ctx := context.WithValue(r.Context(), "request_id", requestID)
        r = r.WithContext(ctx)

        wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
        next.ServeHTTP(wrapped, r)

        slog.Info("request completed",
            slog.String("request_id", requestID),
            slog.String("method", r.Method),
            slog.String("path", r.URL.Path),
            slog.Int("status", wrapped.statusCode),
            slog.Duration("duration", time.Since(start)),
        )
    })
}

// Chain middlewares (chi): applied in the order registered
r := chi.NewRouter()
r.Use(s.loggingMiddleware)
r.With(s.authMiddleware).Get("/protected", s.handleProtected)
```

**Auth Middleware (chi):**
```go
// pkg/auth/middleware.go: validates the Bearer token, stores the user ID in the request context
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        token, err := ExtractTokenFromHeader(r.Header.Get("Authorization"))
        if err != nil {
            m.sendError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing authentication token")
            return
        }

        claims, err := m.validator.ValidateToken(token)
        if err != nil {
            m.sendError(w, http.StatusUnauthorized, "unauthorized", "invalid or missing authentication token")
            return
        }

        ctx := SetUserIDInContext(r.Context(), claims.UserID)
        next.ServeHTTP(w, r.WithContext(ctx))  // Continue to next handler
    })
}

// Usage (services/users/internal/handlers/router.go)
r.Use(logging.ChiRequestLogger(logger))   // request logging + request ID, router-wide
r.Use(sharedmw.Recovery)
r.With(authMiddleware.RequireAuth).Post("/logout-all", authHandler.LogoutAll)  // per-route

// In a handler: userID, err := auth.GetUserIDOrError(r.Context())
```

**Files:**
- `pkg/auth/middleware.go` (JWT auth)
- `pkg/logging/chi_middleware.go` (request logging)
- `pkg/middleware/` (CORS, recovery, rate limiting, security headers)

---

### Context API Pattern (React Native)

**When:** Global state (auth, theme)

**Implementation:**
```typescript
// contexts/AuthContext.tsx
interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  login: () => void;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

export const AuthProvider: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    checkAuthStatus();
  }, []);

  const checkAuthStatus = async () => {
    const authenticated = await AuthService.isAuthenticated();
    if (authenticated) {
      const user = await AuthService.getUser();
      setUser(user);
    }
    setIsLoading(false);
  };

  const logout = async () => {
    await AuthService.logout();
    setUser(null);
  };

  return (
    <AuthContext.Provider value={{ user, isAuthenticated: !!user, isLoading, login, logout }}>
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook with type safety
export const useAuth = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider');
  }
  return context;
};
```

**Usage:**
```typescript
// App.tsx
<AuthProvider>
  <NavigationContainer>
    <RootNavigator />
  </NavigationContainer>
</AuthProvider>

// Any component
const { user, isAuthenticated, logout } = useAuth();
```

**Files:** `apps/mobile/src/contexts/AuthContext.tsx`

---

### Service Class Pattern (React Native)

**When:** API clients, storage abstraction

**Implementation:**
```typescript
// services/auth.ts
export class AuthService {
  private static accessToken: string | null = null;
  private static refreshToken: string | null = null;
  private static user: User | null = null;

  static async initialize(): Promise<void> {
    this.accessToken = await AsyncStorage.getItem(ACCESS_TOKEN_KEY);
    this.refreshToken = await AsyncStorage.getItem(REFRESH_TOKEN_KEY);
    const userJson = await AsyncStorage.getItem(USER_KEY);
    this.user = userJson ? JSON.parse(userJson) : null;
  }

  static async loginWithDevice(): Promise<LoginResponse> {
    const deviceId = await this.getDeviceId();

    const response = await fetch(`${API_BASE_URL}/auth/login/mobile`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ expo_device_id: deviceId }),
    });

    if (!response.ok) {
      throw new Error('Device login failed');
    }

    const data: LoginResponse = await response.json();
    await this.saveTokens({ accessToken: data.access_token, refreshToken: data.refresh_token });
    return data;
  }

  static async refreshAccessToken(): Promise<void> {
    // Implementation
  }

  private static async saveTokens(tokens: AuthTokens): Promise<void> {
    this.accessToken = tokens.accessToken;
    this.refreshToken = tokens.refreshToken;
    await AsyncStorage.setItem(ACCESS_TOKEN_KEY, tokens.accessToken);
    await AsyncStorage.setItem(REFRESH_TOKEN_KEY, tokens.refreshToken);
  }
}
```

**Key Points:**
- Static class for stateless services
- In-memory cache + AsyncStorage persistence
- Typed request/response interfaces

**Files:** `apps/mobile/src/services/auth.ts`

---

## Anti-Patterns to Avoid

### ❌ Global State (Go)

**Don't:**
```go
var DB *sql.DB  // Global database connection

func GetUser(id string) (*User, error) {
    return DB.QueryRow("SELECT * FROM users WHERE id = $1", id)
}
```

**Do:**
```go
type UserRepository struct {
    db *sql.DB
}

func (r *UserRepository) GetUser(ctx context.Context, id string) (*User, error) {
    return r.db.QueryRowContext(ctx, "SELECT * FROM users WHERE id = $1", id)
}
```

---

### ❌ God Objects

**Don't:**
```go
type UserService struct {
    db *sql.DB
}

func (s *UserService) CreateUser(...) error { }
func (s *UserService) SendEmail(...) error { }  // Email logic in user service?
func (s *UserService) ProcessPayment(...) error { }  // Payment logic in user service?
```

**Do:**
```go
type UserService struct {
    userRepo    database.UserRepository
    emailSvc    email.Service
    paymentSvc  payment.Service
}

func (s *UserService) CreateUser(...) error {
    // Create user via repository
    // Delegate email to email service
    // Delegate payment to payment service
}
```

---

### ❌ Leaky Abstractions

**Don't:**
```go
// Repository returns HTTP status code
func (r *UserRepository) GetByEmail(email string) (*User, int, error) {
    // ...
    if err == sql.ErrNoRows {
        return nil, http.StatusNotFound, errors.New("not found")
    }
}
```

**Do:**
```go
// Repository returns domain error
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
    // ...
    if err == sql.ErrNoRows {
        return nil, apperrors.ErrUserNotFound
    }
}

// Handler converts to HTTP
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
    user, err := h.repo.GetUserByID(r.Context(), id)  // id parsed from chi.URLParam(r, "user_id")
    if errors.Is(err, apperrors.ErrUserNotFound) {
        api.WriteError(w, http.StatusNotFound, api.ErrCodeNotFound, "user not found", nil, "v1")
        return
    }
}
```

---

### ❌ Premature Optimization

**Don't:**
```go
// Complex caching layer for data that's rarely accessed
type CachedUserRepository struct {
    cache *redis.Client
    db    *sql.DB
}
```

**Do:**
```go
// Simple repository first, optimize if profiling shows it's a bottleneck
type UserRepository struct {
    db *sql.DB
}
```

**Remember:** "Premature optimization is the root of all evil" - Donald Knuth

---

### ❌ Ignoring Errors

**Don't:**
```go
user, _ := repo.GetByEmail(ctx, email)  // Ignoring error
json.Unmarshal(data, &result)  // Not checking error
```

**Do:**
```go
user, err := repo.GetByEmail(ctx, email)
if err != nil {
    return fmt.Errorf("get user: %w", err)
}

if err := json.Unmarshal(data, &result); err != nil {
    return fmt.Errorf("unmarshal response: %w", err)
}
```

---

### ❌ Magic Numbers

**Don't:**
```typescript
setTimeout(() => retry(), 5000);  // What is 5000?
if (articles.length > 10) { }  // Why 10?
```

**Do:**
```typescript
const RETRY_DELAY_MS = 5000;
const MAX_ARTICLES_PER_PAGE = 10;

setTimeout(() => retry(), RETRY_DELAY_MS);
if (articles.length > MAX_ARTICLES_PER_PAGE) { }
```

---

### ❌ Stringly Typed Code

**Don't:**
```typescript
const sortArticles = (articles: Article[], sortBy: string) => {
  // What are valid values of sortBy? No type safety
}
```

**Do:**
```typescript
type SortOption = 'recent' | 'oldest' | 'title';

const sortArticles = (articles: Article[], sortBy: SortOption) => {
  // Type-safe, auto-complete works
}
```

---

## Conclusion

These principles exist to help you write maintainable, testable, and consistent code. When in doubt:

1. **Follow existing patterns** - Consistency beats perfection
2. **Ask questions** - No question is too simple
3. **Write tests** - They're not optional
4. **Keep it simple** - Complexity is the enemy of reliability

**Living Document:** This guide evolves. If you find a pattern we should document or disagree with a principle, open a discussion.

**Questions?** Reach out in #engineering or create a GitHub discussion.

---

**Document Maintainers:** All senior engineers
**Review Cadence:** Quarterly or when major architectural changes occur
