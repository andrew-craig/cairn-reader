# Docker CI/CD Quick Reference

## Image Naming Convention

All images follow this pattern:
```
ghcr.io/<github-org>/cairn-<service-name>:<tag>
```

Example:
```
ghcr.io/cairn-app/cairn-user-service:latest
ghcr.io/cairn-app/cairn-user-service:v1.0.0
ghcr.io/cairn-app/cairn-user-service:main-abc123
```

## Available Tags

| Tag Pattern | Example | When Created | Use Case |
|-------------|---------|--------------|----------|
| `latest` | `latest` | Push to main | Production deployment |
| `<branch>` | `main`, `develop` | Push to branch | Environment-specific |
| `v<semver>` | `v1.0.0` | Git tag | Versioned releases |
| `<major>.<minor>` | `1.0` | Git tag | Minor version pinning |
| `<major>` | `1` | Git tag | Major version pinning |
| `<branch>-<sha>` | `main-abc123` | Every push | Specific commit deployment |
| `pr-<number>` | `pr-123` | Pull request | PR testing |

## Deployment Examples

### Development Environment (using latest)

```yaml
# infrastructure/docker/docker-compose.yml
services:
  user-service:
    image: ghcr.io/cairn-app/cairn-user-service:develop
    # ... rest of config
```

### Production Environment (using semver)

```yaml
services:
  user-service:
    image: ghcr.io/cairn-app/cairn-user-service:v1.0.0
    # ... rest of config
```

### Testing Specific Commit

```yaml
services:
  user-service:
    image: ghcr.io/cairn-app/cairn-user-service:main-abc123
    # ... rest of config
```

## Using Images Locally

### Pull and run with docker-compose:

```bash
cd infrastructure/docker

# Update docker-compose.yml to use published images
# Then:
docker-compose pull
docker-compose up -d
```

### Pull individual service:

```bash
# Pull latest
docker pull ghcr.io/cairn-app/cairn-user-service:latest

# Run locally
docker run -p 8082:8080 \
  -e DB_HOST=localhost \
  -e DB_PORT=5432 \
  # ... other env vars
  ghcr.io/cairn-app/cairn-user-service:latest
```

## Release Process

### 1. Create a release tag:

```bash
git tag -a v1.0.0 -m "Release version 1.0.0"
git push origin v1.0.0
```

This automatically builds and pushes:
- `ghcr.io/<org>/cairn-user-service:v1.0.0`
- `ghcr.io/<org>/cairn-user-service:1.0`
- `ghcr.io/<org>/cairn-user-service:1`

### 2. Monitor the build:

https://github.com/<org>/cairn/actions

### 3. Verify images:

```bash
# Check all tags for a version
docker manifest inspect ghcr.io/<org>/cairn-user-service:v1.0.0

# Pull and test
docker pull ghcr.io/<org>/cairn-user-service:v1.0.0
```

### 4. Update production deployment:

```bash
cd infrastructure/docker
# Update image tags in docker-compose.yml to v1.0.0
docker-compose pull
docker-compose up -d
```

## Troubleshooting

### "Error: failed to push to ghcr.io"

**Cause:** Missing package write permissions

**Solution:**
1. Go to Settings → Actions → General
2. Under "Workflow permissions", select "Read and write permissions"
3. Re-run the workflow

### "Error: pull access denied"

**Cause:** Not authenticated or package is private

**Solution:**
```bash
# Create a GitHub Personal Access Token with 'read:packages' scope
echo $GITHUB_TOKEN | docker login ghcr.io -u <username> --password-stdin

# Then pull again
docker pull ghcr.io/<org>/cairn-user-service:latest
```

### "Image not found for tag"

**Cause:** Workflow hasn't completed or tag doesn't exist

**Solution:**
1. Check workflow status: https://github.com/<org>/cairn/actions
2. Verify tag exists: https://github.com/<org>/cairn/pkgs/container/cairn-user-service

### Build is slow

**Cause:** Cache not being used

**Solution:**
- First build will be slow (~10 minutes)
- Subsequent builds should be 2-5 minutes with cache
- If still slow, check GitHub Actions cache usage in Settings → Actions → Caches

## Service Matrix

| Service | Dockerfile | Port | Registry Path |
|---------|-----------|------|---------------|
| User Service | services/users/Dockerfile | 8082 | ghcr.io/\<org\>/cairn-user-service |
| Explore Recommender | services/explore/recommender/Dockerfile | 8087 | ghcr.io/\<org\>/cairn-explore-recommender |
| Explore Fetcher | services/explore/fetcher/Dockerfile | 8088 | ghcr.io/\<org\>/cairn-explore-fetcher |
| Content Service | services/read/content/Dockerfile | 8083 | ghcr.io/\<org\>/cairn-content-service |
| Content Worker | services/read/content/Dockerfile.worker | 8084 | ghcr.io/\<org\>/cairn-content-worker |
| Ingest RSS | services/read/fetcher/Dockerfile | 8085 | ghcr.io/\<org\>/cairn-ingest-rss |
| Ingest RSS Worker | services/read/fetcher/Dockerfile.worker | 8086 | ghcr.io/\<org\>/cairn-ingest-rss-worker |

## Advanced Usage

### Build locally with same settings as CI:

```bash
# Install buildx
docker buildx create --use

# Build multi-platform image
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -f services/users/Dockerfile \
  -t ghcr.io/<org>/cairn-user-service:local \
  --load \
  .
```

### Override image in docker-compose:

```bash
# Pull specific version for one service
docker-compose pull user-service

# Override for testing
docker-compose up -d user-service
```

### Check image digest:

```bash
# Get SHA256 digest
docker inspect ghcr.io/<org>/cairn-user-service:v1.0.0 \
  --format='{{index .RepoDigests 0}}'

# Pull by digest (immutable)
docker pull ghcr.io/<org>/cairn-user-service@sha256:abc...
```

## Security Notes

- All images are scanned for vulnerabilities by GitHub
- Artifact attestations are generated for supply chain verification
- Images are signed using GitHub's Sigstore integration
- View security alerts: https://github.com/<org>/cairn/security

## Monitoring & Observability

View build metrics:
- Workflow runs: https://github.com/<org>/cairn/actions
- Package usage: https://github.com/<org>/cairn/pkgs/container/cairn-user-service
- Cache usage: Settings → Actions → Caches

## Next Steps

1. Push code to trigger first build
2. Monitor workflow completion
3. Make packages public (optional): https://github.com/users/<username>/packages
4. Update docker-compose.yml to use published images
5. Test deployment with `docker-compose pull && docker-compose up -d`
