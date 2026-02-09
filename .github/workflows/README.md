# GitHub Actions Workflows

This directory contains GitHub Actions workflows for building and testing Docker images for the Cairn project.

## Workflows

### 1. `docker-build.yml` - Build and Push Docker Images

**Triggers:**
- Push to `main` or `develop` branches
- Version tags (e.g., `v1.0.0`)
- Pull requests to `main` or `develop`
- Manual workflow dispatch

**What it does:**
- Builds all 7 Docker images for Cairn services
- Pushes images to GitHub Container Registry (ghcr.io)
- Creates multi-platform images (amd64 and arm64)
- Tags images with:
  - Branch name (e.g., `main`, `develop`)
  - PR number (e.g., `pr-123`)
  - Semantic version (e.g., `v1.0.0`, `1.0`, `1`)
  - Git SHA (e.g., `main-abc123`)
  - `latest` tag for main branch
- Generates artifact attestations for supply chain security
- Uses GitHub Actions cache for faster builds

**Images built:**
- `ghcr.io/<org>/cairn-user-service`
- `ghcr.io/<org>/cairn-explore-recommender`
- `ghcr.io/<org>/cairn-explore-fetcher`
- `ghcr.io/<org>/cairn-content-service`
- `ghcr.io/<org>/cairn-content-worker`
- `ghcr.io/<org>/cairn-ingest-rss`
- `ghcr.io/<org>/cairn-ingest-rss-worker`

### 2. `docker-test.yml` - Docker Build Test

**Triggers:**
- Pull requests that modify:
  - Service code (`services/**`)
  - Docker infrastructure (`infrastructure/docker/**`)
  - Docker workflows (`.github/workflows/docker-*.yml`)

**What it does:**
- Builds all Docker images to verify they compile correctly
- Does NOT push images to registry
- Tests that images can be instantiated
- Provides fast feedback on PRs

## Setup Instructions

### 1. Enable GitHub Container Registry

The workflows automatically use GitHub Container Registry (ghcr.io). No additional setup is required for public repositories.

For private repositories:
1. Go to your repository Settings → Actions → General
2. Under "Workflow permissions", ensure "Read and write permissions" is selected
3. Check "Allow GitHub Actions to create and approve pull requests"

### 2. Configure Package Visibility (Optional)

By default, packages are private. To make them public:
1. Go to https://github.com/users/<username>/packages
2. Click on the package (e.g., `cairn-user-service`)
3. Click "Package settings"
4. Scroll to "Danger Zone"
5. Click "Change visibility" → "Public"

### 3. Using the Images

#### Pull images from GitHub Container Registry:

```bash
# Pull latest version
docker pull ghcr.io/<org>/cairn-user-service:latest

# Pull specific version
docker pull ghcr.io/<org>/cairn-user-service:v1.0.0

# Pull specific commit
docker pull ghcr.io/<org>/cairn-user-service:main-abc123
```

#### Update docker-compose.yml to use published images:

```yaml
services:
  user-service:
    image: ghcr.io/<org>/cairn-user-service:latest
    # Remove the 'build' section
    ports:
      - "8082:8080"
    environment:
      # ... same as before
```

#### Authenticate Docker (for private packages):

```bash
# Create a Personal Access Token (classic) with 'read:packages' scope
# Then login:
echo $GITHUB_TOKEN | docker login ghcr.io -u <username> --password-stdin
```

## Triggering Builds Manually

You can manually trigger a build from the GitHub UI:
1. Go to Actions → Build and Push Docker Images
2. Click "Run workflow"
3. Select branch and click "Run workflow"

## Monitoring Builds

- View workflow runs: https://github.com/<org>/cairn/actions
- Each matrix job builds one service in parallel
- Build time: ~5-10 minutes for all services (parallel)
- Check logs for individual service builds

## Customization

### Building for Different Platforms

The workflow builds for `linux/amd64` and `linux/arm64` by default. To change:

```yaml
platforms: linux/amd64,linux/arm64,linux/arm/v7
```

### Adding New Services

To add a new service to the build matrix:

```yaml
- service: new-service
  dockerfile: services/new-service/Dockerfile
  context: .
```

### Changing Image Tags

Modify the `tags:` section in `docker-build.yml`:

```yaml
tags: |
  type=ref,event=branch
  type=semver,pattern={{version}}
  # Add custom tags here
```

## Caching

The workflows use GitHub Actions cache to speed up builds:
- First build: ~10 minutes
- Subsequent builds: ~2-5 minutes (with cache)
- Cache is shared across workflows and branches

## Security

- Images are scanned for vulnerabilities (via GitHub's built-in scanning)
- Artifact attestations provide supply chain security
- SBOM (Software Bill of Materials) can be generated on request
- Images are signed with GitHub's Sigstore integration

## Troubleshooting

### Build fails with "permission denied"
- Check workflow permissions in Settings → Actions → General
- Ensure GITHUB_TOKEN has write permissions

### Image not found when pulling
- Check package visibility settings
- Verify you're authenticated with `docker login ghcr.io`
- Ensure the workflow completed successfully

### Cache issues
- Clear cache by re-running workflow with "Re-run all jobs"
- Or modify workflow to use `cache-from: type=gha,mode=min`

## iOS Deployment Workflows

### 🚀 Recommended: Fastlane (Free)

**File:** `ios-testflight-fastlane.yml`

- ✅ **Free** (uses GitHub Actions macOS runners)
- ✅ **Industry standard** for iOS deployment
- ✅ **Full control** over build process
- ⚙️ **Setup:** Follow [apps/mobile/FASTLANE_SETUP.md](../../apps/mobile/FASTLANE_SETUP.md)

**Triggers:**
- Manual: Actions tab → "iOS TestFlight (Fastlane)" → Run workflow
- Automatic: Push a tag like `mobile-v1.0.0`

**What it does:**
- Generates native iOS code with `expo prebuild`
- Installs dependencies and CocoaPods
- Downloads code signing certificates via Fastlane Match
- Builds the app with Xcode
- Auto-increments build number
- Uploads to TestFlight

### 💰 Alternative: EAS Build (Easiest)

**File:** `ios-testflight.yml`

- 💵 **Paid** service ($29/month after free tier)
- ✅ **Easiest setup** (minimal configuration)
- ✅ **Managed** by Expo
- ⚙️ **Setup:** Follow [TESTFLIGHT_SETUP.md](TESTFLIGHT_SETUP.md)

**Triggers:**
- Manual: Actions tab → "iOS TestFlight Deployment" → Run workflow

### 🔧 Advanced: Native Xcode Build

**File:** `ios-testflight-local-build.yml`

- ✅ **Free** but complex
- ⚙️ **Maximum control** over every build step
- 🛠️ **Manual** certificate management required

### Comparison of iOS Options

See [BUILD_OPTIONS.md](BUILD_OPTIONS.md) for a detailed comparison of all iOS build approaches.

## Related Documentation

- [Docker Compose Setup](/infrastructure/docker/README.md)
- [Service Documentation](/CLAUDE.md)
- [GitHub Container Registry Docs](https://docs.github.com/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
- [Fastlane Setup Guide](../../apps/mobile/FASTLANE_SETUP.md) - iOS deployment setup
- [iOS Build Options Comparison](BUILD_OPTIONS.md) - Compare EAS vs Fastlane vs Native
