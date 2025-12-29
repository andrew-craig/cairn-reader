# Read Service Restructuring Checklist

Use this checklist to track progress during the restructuring process.

## Phase 1: Directory Reorganization
- [ ] Move `services/content-service/` to `content/`
- [ ] Move `services/rss-fetcher-service/` to `fetcher/`
- [ ] Remove empty `services/` directory
- [ ] Rename `fetcher/cmd/server/` to `fetcher/cmd/fetcher/`
- [ ] Create `content/cmd/content/` for main API server (if needed)
- [ ] Create `bin/` directory for build artifacts

## Phase 2: Module and Import Path Updates
- [ ] Create service-level `go.mod` at `services/read/go.mod`
- [ ] Merge dependencies from both services into new go.mod
- [ ] Remove `content/go.mod` and `content/go.sum`
- [ ] Remove `fetcher/go.mod` and `fetcher/go.sum`
- [ ] Update all imports in `content/` from `cairn-read/services/content-service` to `cairn-read/content`
- [ ] Update all imports in `fetcher/` from `cairn-read/services/rss-fetcher-service` to `cairn-read/fetcher`
- [ ] Run `go mod tidy` to verify

## Phase 3: Extract Shared Code to pkg/
- [ ] Create `pkg/logging/` directory
- [ ] Create `pkg/models/` directory (if needed)
- [ ] Extract common logging setup to `pkg/logging/logger.go`
- [ ] Update content service to use `pkg/logging`
- [ ] Update fetcher service to use `pkg/logging`
- [ ] Move any shared models to `pkg/models/`

## Phase 4: Update Configuration Files
- [ ] Update `docker-compose.yml` service build paths
- [ ] Update `docker-compose.yml` service names (if needed)
- [ ] Update `Makefile` build targets for new structure
- [ ] Update `Makefile` test targets for new structure
- [ ] Update `content/Dockerfile` COPY paths
- [ ] Update `content/Dockerfile.worker` COPY paths
- [ ] Update `fetcher/Dockerfile` COPY paths
- [ ] Update `fetcher/Dockerfile.worker` COPY paths
- [ ] Review and update `.env.example` files

## Phase 5: Update Documentation
- [ ] Update `README.md` with new directory structure
- [ ] Update `README.md` with correct build commands
- [ ] Update `README.md` with correct module paths
- [ ] Update `content/README.md` (if exists)
- [ ] Update `fetcher/API_REFERENCE.md` (if exists)
- [ ] Update `IMPLEMENTATION_PLAN.md` with new paths
- [ ] Update `INTEGRATION_TESTS.md` with new paths
- [ ] Update `docs/ARCHITECTURE.md` with new paths
- [ ] Update `docs/DEPLOYMENT.md` with new paths
- [ ] Update `docs/CONFIGURATION.md` with new paths
- [ ] Update `docs/TROUBLESHOOTING.md` with new paths

## Phase 6: Testing and Verification
- [ ] Run `go mod tidy` successfully
- [ ] Run `go mod verify` successfully
- [ ] Build content service: `make build-content`
- [ ] Build content worker: `make build-content-worker`
- [ ] Build fetcher service: `make build-fetcher`
- [ ] Build fetcher worker: `make build-fetcher-worker`
- [ ] Run content service tests: `make test-content`
- [ ] Run fetcher service tests: `make test-fetcher`
- [ ] Run integration tests: `./scripts/run-integration-tests.sh`
- [ ] Test Docker Compose build: `docker-compose build`
- [ ] Test Docker Compose up: `docker-compose up`
- [ ] Verify all services start correctly
- [ ] Test content service API endpoints
- [ ] Test fetcher service API endpoints
- [ ] Verify database connections
- [ ] Test fetcher → content service communication
- [ ] Stop Docker Compose: `docker-compose down`

## Phase 7: Clean Up
- [ ] Remove redundant README files (if any)
- [ ] Consolidate `.gitignore` files
- [ ] Remove test database files
- [ ] Clean up any `.env` or temp files
- [ ] Update root `.gitignore` for read service artifacts
- [ ] Run `git status` to verify no untracked sensitive files

## Post-Restructuring Tasks
- [ ] Update repository root `CLAUDE.md` with Read service documentation
- [ ] Commit all changes with descriptive message
- [ ] Push to branch
- [ ] Create pull request
- [ ] Include before/after structure in PR description
- [ ] Update CI/CD pipelines (if applicable)
- [ ] Verify cross-service integration
- [ ] Test authentication integration (if applicable)
- [ ] Run end-to-end workflow tests

## Verification Commands

After each phase, use these commands to verify:

```bash
# Module verification
cd services/read
go mod tidy
go mod verify

# Build verification
make build-all

# Test verification
make test-all

# Docker verification
docker-compose build
docker-compose up -d
docker-compose ps
docker-compose logs
docker-compose down
```

## Rollback Plan

If issues occur:

1. **Before committing:** Use `git status` and `git diff` to review changes
2. **After committing:** Use `git reset --hard HEAD~1` to undo last commit
3. **After pushing:** Create a revert commit or restore from backup

## Notes

- Take backups before starting major changes
- Test after each phase before proceeding
- Document any deviations from the plan
- Update this checklist as you progress

---

**Started:** _______________
**Completed:** _______________
**Issues encountered:**
-
-
-
