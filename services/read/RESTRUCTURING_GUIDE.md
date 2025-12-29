# Read Service Restructuring Guide

This guide provides step-by-step instructions for restructuring the Read service to match the main cairn repository conventions, based on the patterns established in the Explore and Users services.

## Current Structure Issues

The Read service files were merged from the `cairn-read` repository with the following non-standard structure:

```
services/read/
├── services/                    # ❌ Unnecessary nesting
│   ├── content-service/         # ❌ Should be content/
│   └── rss-fetcher-service/     # ❌ Should be fetcher/
├── docs/
├── scripts/
├── docker-compose.yml
├── Makefile
└── requirements.md
```

## Target Structure

The restructured Read service should follow the Explore service pattern:

```
services/read/
├── content/                     # ✅ Content storage microservice
│   ├── cmd/
│   │   ├── content/            # Main API server
│   │   │   └── main.go
│   │   └── worker/             # Background job worker
│   │       └── main.go
│   ├── internal/               # Private implementation
│   │   ├── api/
│   │   ├── database/
│   │   ├── jobs/
│   │   ├── models/
│   │   ├── processor/
│   │   ├── repository/
│   │   ├── service/
│   │   └── testhelpers/
│   ├── migrations/             # Database migrations
│   ├── Dockerfile
│   ├── Dockerfile.worker
│   ├── integration_test.go
│   └── README.md
├── fetcher/                    # ✅ RSS feed fetcher microservice
│   ├── cmd/
│   │   ├── fetcher/           # Main API server
│   │   │   └── main.go
│   │   └── worker/            # Background feed worker
│   │       └── main.go
│   ├── internal/              # Private implementation
│   │   ├── api/
│   │   ├── client/
│   │   ├── database/
│   │   ├── fetcher/
│   │   ├── jobs/
│   │   ├── models/
│   │   ├── processor/
│   │   ├── repository/
│   │   ├── scheduler/
│   │   ├── service/
│   │   ├── testhelpers/
│   │   └── worker/
│   ├── migrations/            # Database migrations
│   ├── api/                   # OpenAPI specs
│   │   └── openapi.yaml
│   ├── Dockerfile
│   ├── Dockerfile.worker
│   ├── integration_test.go
│   ├── API_REFERENCE.md
│   └── README.md (optional, can be removed if redundant)
├── pkg/                       # ✅ Shared packages across both services
│   ├── logging/              # Centralized logging setup
│   └── models/               # Shared data models (if any)
├── bin/                       # ✅ Build artifacts directory
├── docs/                      # ✅ Service-level documentation
│   ├── ARCHITECTURE.md
│   ├── CONFIGURATION.md
│   ├── DEPLOYMENT.md
│   └── TROUBLESHOOTING.md
├── scripts/                   # ✅ Helper scripts
│   └── run-integration-tests.sh
├── go.mod                     # ✅ Service-level Go module
├── go.sum
├── docker-compose.yml         # ✅ Orchestration for all read microservices
├── Makefile                   # ✅ Build commands for all microservices
├── requirements.md            # ✅ Detailed requirements
├── IMPLEMENTATION_PLAN.md
├── INTEGRATION_TESTS.md
├── PHASE_5_5_PRIORITY_6_SUMMARY.md
└── README.md                  # ✅ Service overview
```

## Restructuring Steps

### Phase 1: Directory Reorganization

**1.1 Rename and move content-service**
```bash
cd services/read
mv services/content-service content
```

**1.2 Rename and move rss-fetcher-service**
```bash
mv services/rss-fetcher-service fetcher
```

**1.3 Remove empty services directory**
```bash
rmdir services
```

**1.4 Reorganize content service commands**
```bash
# The content service currently has cmd/worker/
# It needs cmd/content/ for the main API server (currently missing)
# Check if there's a server command to create/move
```

**1.5 Reorganize fetcher service commands**
```bash
cd fetcher
# Current: cmd/worker/ and cmd/server/
# Should be: cmd/worker/ and cmd/fetcher/
mv cmd/server cmd/fetcher
```

### Phase 2: Module and Import Path Updates

**2.1 Create service-level go.mod**

Create a new `services/read/go.mod`:
```go
module github.com/andrew-craig/cairn-read

go 1.24.0

require (
    github.com/go-chi/chi/v5 v5.2.3
    github.com/go-shiori/go-readability v0.0.0-20251205110129-5db1dc9836f0
    github.com/google/uuid v1.6.0
    github.com/lib/pq v1.10.9
    github.com/microcosm-cc/bluemonday v1.0.27
    github.com/mmcdole/gofeed v1.3.0
    github.com/robfig/cron/v3 v3.0.1
    github.com/sony/gobreaker v0.5.0
    github.com/stretchr/testify v1.11.1
    go.uber.org/zap v1.27.1
)

// Additional dependencies from both services...
```

**2.2 Remove individual service go.mod files**
```bash
rm content/go.mod content/go.sum
rm fetcher/go.mod fetcher/go.sum
```

**2.3 Update all import paths**

Change all imports from:
- `github.com/andrew-craig/cairn-read/services/content-service/...` → `github.com/andrew-craig/cairn-read/content/...`
- `github.com/andrew-craig/cairn-read/services/rss-fetcher-service/...` → `github.com/andrew-craig/cairn-read/fetcher/...`

This requires updating:
- All `.go` files in `content/`
- All `.go` files in `fetcher/`
- Test files
- Integration tests

**Example automated replacement:**
```bash
# For content service
find content -name "*.go" -type f -exec sed -i 's|github.com/andrew-craig/cairn-read/services/content-service|github.com/andrew-craig/cairn-read/content|g' {} +

# For fetcher service
find fetcher -name "*.go" -type f -exec sed -i 's|github.com/andrew-craig/cairn-read/services/rss-fetcher-service|github.com/andrew-craig/cairn-read/fetcher|g' {} +
```

### Phase 3: Extract Shared Code to pkg/

**3.1 Create pkg directory structure**
```bash
mkdir -p pkg/logging
mkdir -p pkg/models
```

**3.2 Extract shared logging setup**

If both services use similar logging configuration (zap, slog), extract to `pkg/logging/`:
- Create `pkg/logging/logger.go` with common logging setup
- Update both services to import from `github.com/andrew-craig/cairn-read/pkg/logging`

**3.3 Extract shared models (if any)**

If there are data models shared between content and fetcher services:
- Move to `pkg/models/`
- Update imports

**Note:** Based on the Explore service pattern, `pkg/logging` should provide a `NewLogger()` function and `SetDefault()` for consistent logging across all microservices.

### Phase 4: Update Configuration Files

**4.1 Update docker-compose.yml**

Update service names and paths:
```yaml
services:
  content-service:
    build:
      context: .
      dockerfile: content/Dockerfile
    # ...

  content-worker:
    build:
      context: .
      dockerfile: content/Dockerfile.worker
    # ...

  fetcher-service:
    build:
      context: .
      dockerfile: fetcher/Dockerfile
    # ...

  fetcher-worker:
    build:
      context: .
      dockerfile: fetcher/Dockerfile.worker
    # ...
```

**4.2 Update Makefile**

Update targets to reflect new paths:
```makefile
# Build targets
.PHONY: build-content
build-content:
	cd content && go build -o ../bin/content ./cmd/content

.PHONY: build-content-worker
build-content-worker:
	cd content && go build -o ../bin/content-worker ./cmd/worker

.PHONY: build-fetcher
build-fetcher:
	cd fetcher && go build -o ../bin/fetcher ./cmd/fetcher

.PHONY: build-fetcher-worker
build-fetcher-worker:
	cd fetcher && go build -o ../bin/fetcher-worker ./cmd/worker

# Test targets
.PHONY: test-content
test-content:
	cd content && go test ./...

.PHONY: test-fetcher
test-fetcher:
	cd fetcher && go test ./...

.PHONY: test-all
test-all: test-content test-fetcher
```

**4.3 Update Dockerfiles**

Update Dockerfile build contexts from:
```dockerfile
COPY services/content-service/ .
```

To:
```dockerfile
COPY content/ ./content/
COPY pkg/ ./pkg/
COPY go.mod go.sum ./
```

**4.4 Update .env.example files**

Ensure environment variable examples reflect the new structure.

### Phase 5: Update Documentation

**5.1 Update README.md**

Update the main `services/read/README.md` to reflect:
- New directory structure
- Updated build commands
- Correct module paths
- Updated development workflow

**5.2 Update service-specific READMEs**

Update `content/README.md` and `fetcher/API_REFERENCE.md` (if they exist) with:
- Correct import paths
- Updated command examples
- New directory references

**5.3 Update IMPLEMENTATION_PLAN.md and other docs**

Update any references to old paths in:
- IMPLEMENTATION_PLAN.md
- INTEGRATION_TESTS.md
- docs/*.md files

### Phase 6: Testing and Verification

**6.1 Verify module structure**
```bash
cd services/read
go mod tidy
go mod verify
```

**6.2 Build all services**
```bash
make build-all
# Or individually:
make build-content
make build-content-worker
make build-fetcher
make build-fetcher-worker
```

**6.3 Run tests**
```bash
make test-all
# Or individually:
make test-content
make test-fetcher
```

**6.4 Run integration tests**
```bash
./scripts/run-integration-tests.sh
```

**6.5 Test Docker Compose**
```bash
docker-compose up --build
# Verify all services start correctly
# Test API endpoints
# Check database connections
docker-compose down
```

### Phase 7: Clean Up

**7.1 Remove redundant files**
- Consider consolidating README files if there's duplication
- Remove any leftover `.gitignore` files in subdirectories (keep only root-level)
- Clean up any `.env` or test database files

**7.2 Update .gitignore**

Ensure services/read/.gitignore includes:
```
# Build artifacts
bin/
*.exe
*.dll
*.so
*.dylib

# Test artifacts
*.test
*.out
coverage.html

# IDE
.vscode/
.idea/

# Environment
.env
*.local

# Database
*.db
```

**7.3 Verify git status**
```bash
git status
# Ensure no untracked build artifacts or sensitive files
```

## Comparison with Existing Services

### Explore Service Pattern (Reference)
```
services/explore/
├── fetcher/              # One microservice
├── recommender/          # Another microservice
├── pkg/                  # Shared code
│   ├── logging/
│   └── models/
├── go.mod                # Single module for entire service
└── docker-compose.yml
```

### Users Service Pattern (Reference)
```
services/users/
├── cmd/                  # Single service, multiple commands
│   ├── migrate/
│   └── user-service/
├── internal/
├── pkg/
├── migrations/
├── go.mod
└── Dockerfile
```

### Read Service Pattern (After Restructuring)
```
services/read/
├── content/              # Content microservice
├── fetcher/              # Fetcher microservice
├── pkg/                  # Shared code
│   └── logging/
├── go.mod                # Single module for entire service
└── docker-compose.yml
```

## Key Conventions to Follow

Based on analysis of Explore and Users services:

1. **Module Naming:**
   - Service level: `github.com/andrew-craig/cairn-{service}`
   - Example: `github.com/andrew-craig/cairn-read`

2. **Import Paths:**
   - Within service: `github.com/andrew-craig/cairn-{service}/{microservice}/internal/{package}`
   - Example: `github.com/andrew-craig/cairn-read/content/internal/api`

3. **Command Structure:**
   - Main entry point: `cmd/{service-name}/main.go`
   - Workers: `cmd/worker/main.go`
   - Utilities: `cmd/{utility-name}/main.go`

4. **Shared Code:**
   - Create `pkg/` at service level for code shared between microservices
   - Common packages: `pkg/logging`, `pkg/models`

5. **Documentation:**
   - Service-level README: High-level overview, getting started
   - docs/ directory: Detailed architecture, deployment, troubleshooting
   - Microservice-level README (optional): Only if significantly different from service-level

6. **Testing:**
   - Unit tests: alongside code (`_test.go` files)
   - Integration tests: `integration_test.go` at microservice root
   - Test helpers: `internal/testhelpers/`

7. **Build Artifacts:**
   - Output to `bin/` directory at service level
   - Ignored in git via `.gitignore`

8. **Docker:**
   - One Dockerfile per deployable component
   - docker-compose.yml at service level orchestrates all microservices
   - Multi-stage builds for smaller production images

## Post-Restructuring Tasks

After completing the restructuring:

1. **Update CLAUDE.md** at repository root:
   - Document the new Read service structure
   - Add common commands for building/testing
   - Update API endpoint documentation
   - Document the Content and Fetcher microservices

2. **Create Pull Request:**
   - Commit all changes with descriptive message
   - Create PR for review
   - Include before/after directory structure in PR description

3. **Update CI/CD** (if applicable):
   - Update build pipelines to use new paths
   - Update test runners
   - Update deployment scripts

4. **Verify Cross-Service Integration:**
   - Test Fetcher → Content API communication
   - Verify authentication integration (if applicable)
   - Test end-to-end workflows

## Automated Restructuring Script

For convenience, here's a complete bash script to automate most of the restructuring:

```bash
#!/bin/bash
set -e

cd services/read

echo "Phase 1: Reorganizing directories..."
mv services/content-service content
mv services/rss-fetcher-service fetcher
rmdir services

echo "Phase 2: Renaming commands..."
mv fetcher/cmd/server fetcher/cmd/fetcher

echo "Phase 3: Creating shared pkg directory..."
mkdir -p pkg/logging
mkdir -p pkg/models

echo "Phase 4: Updating import paths..."
find content -name "*.go" -type f -exec sed -i 's|github.com/andrew-craig/cairn-read/services/content-service|github.com/andrew-craig/cairn-read/content|g' {} +
find fetcher -name "*.go" -type f -exec sed -i 's|github.com/andrew-craig/cairn-read/services/rss-fetcher-service|github.com/andrew-craig/cairn-read/fetcher|g' {} +

echo "Phase 5: Removing individual go.mod files..."
rm -f content/go.mod content/go.sum
rm -f fetcher/go.mod fetcher/go.sum

echo "Phase 6: Creating bin directory..."
mkdir -p bin

echo "Done! Next steps:"
echo "1. Create services/read/go.mod with merged dependencies"
echo "2. Update docker-compose.yml service paths"
echo "3. Update Makefile build targets"
echo "4. Update Dockerfiles with new COPY paths"
echo "5. Run 'go mod tidy' to verify module structure"
echo "6. Test builds and integration tests"
```

## Troubleshooting

**Import cycles:**
- If you encounter import cycles after restructuring, review your `pkg/` extraction
- Ensure `pkg/` packages don't import from `internal/`

**Module resolution errors:**
- Run `go mod tidy` at the service root (`services/read/`)
- Clear Go build cache: `go clean -modcache`
- Verify all import paths are updated correctly

**Docker build failures:**
- Ensure Dockerfiles copy the correct paths (including `pkg/`)
- Update `.dockerignore` if needed
- Verify build context is set correctly in docker-compose.yml

**Test failures:**
- Update test helper import paths
- Verify database migration paths in tests
- Check environment variable references

## References

- Explore service: `services/explore/`
- Users service: `services/users/`
- Go modules documentation: https://go.dev/ref/mod
- Docker Compose: https://docs.docker.com/compose/
