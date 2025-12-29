# Read Service Restructuring Summary

Quick reference for restructuring the Read service to match cairn repository conventions.

## Problem

The Read service was merged from `cairn-read` repository with non-standard structure:
- Services nested too deep: `services/read/services/content-service/`
- Inconsistent naming: `content-service` vs `content`, `rss-fetcher-service` vs `fetcher`
- Separate go.mod files instead of unified module
- No shared `pkg/` directory for common code

## Solution

Restructure to match the Explore service pattern used in cairn repository.

## Quick Start

Run the automated restructuring script:

```bash
cd services/read

# 1. Run automated restructuring
./restructure.sh

# 2. Create unified go.mod (manual step)
# Copy dependencies from content/go.mod and fetcher/go.mod
cat > go.mod << 'EOF'
module github.com/andrew-craig/cairn-read

go 1.24.0
# ... add merged dependencies
EOF

# 3. Tidy and verify
go mod tidy
go mod verify

# 4. Update docker-compose.yml and Makefile (see guide)

# 5. Test
make build-all
make test-all
docker-compose up --build
```

## Key Changes

### Directory Structure

**Before:**
```
services/read/
└── services/
    ├── content-service/
    └── rss-fetcher-service/
```

**After:**
```
services/read/
├── content/        # Content storage microservice
├── fetcher/        # RSS fetcher microservice
├── pkg/           # Shared code (logging, models)
├── bin/           # Build artifacts
└── go.mod         # Unified module
```

### Module Paths

**Before:**
- `github.com/andrew-craig/cairn-read/services/content-service`
- `github.com/andrew-craig/cairn-read/services/rss-fetcher-service`

**After:**
- `github.com/andrew-craig/cairn-read/content`
- `github.com/andrew-craig/cairn-read/fetcher`
- `github.com/andrew-craig/cairn-read/pkg/logging`

### Command Structure

**Before:**
- `content-service/cmd/worker/`
- `rss-fetcher-service/cmd/server/`
- `rss-fetcher-service/cmd/worker/`

**After:**
- `content/cmd/content/` (main API server)
- `content/cmd/worker/` (background jobs)
- `fetcher/cmd/fetcher/` (main API server)
- `fetcher/cmd/worker/` (background feed processing)

## Automated Script

Save as `services/read/restructure.sh`:

```bash
#!/bin/bash
set -e

echo "Starting Read service restructuring..."

# Phase 1: Move directories
echo "Moving directories..."
mv services/content-service content
mv services/rss-fetcher-service fetcher
rmdir services

# Phase 2: Rename commands
echo "Renaming commands..."
mv fetcher/cmd/server fetcher/cmd/fetcher

# Phase 3: Create shared directories
echo "Creating shared directories..."
mkdir -p pkg/logging
mkdir -p pkg/models
mkdir -p bin

# Phase 4: Update import paths
echo "Updating import paths in content service..."
find content -name "*.go" -type f -exec sed -i \
  's|github.com/andrew-craig/cairn-read/services/content-service|github.com/andrew-craig/cairn-read/content|g' {} +

echo "Updating import paths in fetcher service..."
find fetcher -name "*.go" -type f -exec sed -i \
  's|github.com/andrew-craig/cairn-read/services/rss-fetcher-service|github.com/andrew-craig/cairn-read/fetcher|g' {} +

# Phase 5: Backup and remove individual go.mod files
echo "Backing up go.mod files..."
cp content/go.mod content-go.mod.bak
cp fetcher/go.mod fetcher-go.mod.bak
rm -f content/go.mod content/go.sum
rm -f fetcher/go.mod fetcher/go.sum

echo "Restructuring complete!"
echo ""
echo "Next manual steps:"
echo "1. Create services/read/go.mod from content-go.mod.bak and fetcher-go.mod.bak"
echo "2. Run: go mod tidy"
echo "3. Update docker-compose.yml paths"
echo "4. Update Makefile targets"
echo "5. Update Dockerfile COPY statements"
echo "6. Run: make build-all"
echo "7. Run: make test-all"
echo ""
echo "See RESTRUCTURING_GUIDE.md for detailed instructions."
```

## Manual Steps Required

After running the automated script:

### 1. Create Unified go.mod

```bash
cd services/read

# Merge dependencies from backups
cat content-go.mod.bak fetcher-go.mod.bak

# Create new go.mod with merged dependencies
nano go.mod  # or your preferred editor
```

### 2. Update docker-compose.yml

```yaml
services:
  content-service:
    build:
      context: .
      dockerfile: content/Dockerfile  # Updated path
    # ...

  fetcher-service:
    build:
      context: .
      dockerfile: fetcher/Dockerfile  # Updated path
    # ...
```

### 3. Update Makefile

```makefile
build-content:
	cd content && go build -o ../bin/content ./cmd/content

build-fetcher:
	cd fetcher && go build -o ../bin/fetcher ./cmd/fetcher

test-content:
	cd content && go test ./...

test-fetcher:
	cd fetcher && go test ./...
```

### 4. Update Dockerfiles

```dockerfile
# In content/Dockerfile and fetcher/Dockerfile
COPY content/ ./content/   # or fetcher/ ./fetcher/
COPY pkg/ ./pkg/
COPY go.mod go.sum ./
```

## Verification

```bash
# Module check
cd services/read
go mod tidy
go mod verify

# Build check
make build-all

# Test check
make test-all

# Docker check
docker-compose up --build
docker-compose ps  # All services should be "Up"
docker-compose down
```

## References

- **Full Guide:** `RESTRUCTURING_GUIDE.md` (567 lines, comprehensive)
- **Checklist:** `RESTRUCTURING_CHECKLIST.md` (track progress)
- **Example:** `services/explore/` (reference implementation)

## Support

If you encounter issues:

1. Check `RESTRUCTURING_GUIDE.md` troubleshooting section
2. Review `services/explore/` for reference patterns
3. Verify all import paths updated correctly
4. Clear build cache: `go clean -modcache`
5. Rebuild Docker images: `docker-compose build --no-cache`

## Time Estimate

- Automated script: 2 minutes
- Manual steps: 30-60 minutes
- Testing and verification: 30 minutes
- Documentation updates: 30 minutes

**Total: 2-3 hours**
