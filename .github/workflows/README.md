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
- `ghcr.io/<org>/cairn-content-service`
- `ghcr.io/<org>/cairn-content-worker`
- `ghcr.io/<org>/cairn-ingest-rss`
- `ghcr.io/<org>/cairn-ingest-rss-worker`

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

#### Update prod/docker-compose.yml to use published images:

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

## iOS Workflows

iOS ships only from the native app in `apps/ios`. The Expo app in `apps/mobile` is Android-only and has no release workflow.

### `ios-checks.yml` - Build and test

**Triggers:** pull requests to `main` that touch `apps/ios/**`.

Builds `CairnReader.xcodeproj` and runs its tests on an iOS 26 simulator (`macos-26` runner).

### `ios-native-testflight.yml` - TestFlight release

**Triggers:**
- Manual: Actions tab → "iOS Native TestFlight" → Run workflow
- Automatic: push a tag like `ios-v1.1.0`

**What it does:**
- Runs `bundle exec fastlane beta` in `apps/ios` (no Node, `expo prebuild` or CocoaPods)
- Downloads signing certificates and profiles via Fastlane Match (read-only)
- Builds `CairnReader.xcodeproj`, build number = latest TestFlight + 1
- Uploads to TestFlight

**Secrets:** `ASC_KEY_ID`, `ASC_ISSUER_ID`, `ASC_KEY_CONTENT`, `FASTLANE_APPLE_ID`, `MATCH_GIT_URL`, `MATCH_PASSWORD`, `MATCH_GIT_BASIC_AUTHORIZATION`, `TEAM_ID`. The marketing version is `MARKETING_VERSION` in the Xcode project. The `build_only` lane does the signed build without uploading.

**One-time provisioning:** the App Group `group.com.cairnapp.cairnreader` must be enabled on the App ID `com.cairnapp.cairnreader` in the Apple Developer portal. Then regenerate the Match profiles once, from a Mac with the Match secrets exported (`MATCH_GIT_URL`, `MATCH_PASSWORD`, `FASTLANE_APPLE_ID`, `ASC_*`):

```bash
cd apps/ios && bundle install && bundle exec fastlane match appstore --force
```

CI runs Match read-only, so `beta` fails to sign until this is done.

## Mobile (Expo, Android) Workflow

### `mobile-checks.yml`

**Triggers:** pull requests to `main` that touch `apps/mobile/**`, `apps/shared/**` or the root npm manifests.

Runs TypeScript, ESLint, Jest, knip and an Android Metro bundle export (`expo export --platform android`).

## Related Documentation

- [Docker Compose Setup](/infrastructure/docker/README.md)
- [Service Documentation](/CLAUDE.md)
- [iOS app](/apps/ios/CLAUDE.md) - native app, Fastlane lanes
- [GitHub Container Registry Docs](https://docs.github.com/packages/working-with-a-github-packages-registry/working-with-the-container-registry)
