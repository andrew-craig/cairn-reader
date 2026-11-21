# GitHub Actions Workflows

This directory contains the CI and Docker image workflows for the Cairn project.

## Docker image workflows

There is one workflow per image (`docker-build-<name>.yml`), each building from its own Dockerfile and pushing to GitHub Container Registry as `ghcr.io/<org>/cairn-<name>`.

| Workflow | Image | Dockerfile |
|----------|-------|------------|
| `docker-build-user-service.yml` | `cairn-user-service` | `services/users/Dockerfile` |
| `docker-build-content-service.yml` | `cairn-content-service` | `services/read/content/Dockerfile` |
| `docker-build-content-worker.yml` | `cairn-content-worker` | `services/read/content/Dockerfile.worker` |
| `docker-build-ingest-rss.yml` | `cairn-ingest-rss` | `services/read/fetcher/Dockerfile` |
| `docker-build-ingest-rss-worker.yml` | `cairn-ingest-rss-worker` | `services/read/fetcher/Dockerfile.worker` |
| `docker-build-ingest-email.yml` | `cairn-ingest-email` | `services/read/email/Dockerfile` |
| `docker-build-ingest-email-worker.yml` | `cairn-ingest-email-worker` | `services/read/email/Dockerfile.worker` |
| `docker-build-web.yml` | `cairn-web` | `apps/web/Dockerfile` |
| `docker-build-selfhost.yml` | `cairn-selfhost` | `infrastructure/docker/selfhost/Dockerfile` |

**Triggers:**
- Push to `main` or a version tag (`v*`), when files under the image's paths change
- Pull requests to `main` (build only, nothing is pushed)
- Manual workflow dispatch

**What they do:**
- Build for `linux/arm64` only
- Tag images with the branch name, PR number, semantic version (`1.2.3`, `1.2`, `1`), Git SHA, and `latest` on the default branch
- Use the GitHub Actions cache (`type=gha`)

The prod stack (`infrastructure/docker/prod/docker-compose.yml`) pulls these images. See [docs/DEPLOYMENT.md](/docs/DEPLOYMENT.md).

## Other checks

- `go-checks.yml` (Go Checks): formatting, `go vet`, builds, tests and integration tests for `cmd/`, `services/` and `pkg/`
- `web-checks.yml` (Web App Checks): `tsc`, ESLint and Vitest for `apps/web` and `apps/shared`
- `design-tokens.yml` (Design Tokens): fails if the generated color files differ from `design/tokens/colors.json`, see [design/tokens/README.md](/design/tokens/README.md)
- `selfhost-compose-smoke.yml` (Selfhost Compose Smoke Test): builds the selfhost image and smoke-tests its Compose setup

## Setup

The workflows use GitHub Container Registry with the repository's `GITHUB_TOKEN`. For private repositories, under Settings, Actions, General, set "Workflow permissions" to "Read and write permissions". Packages are private by default; change visibility under the package's settings.

To pull a private image, authenticate first:

```bash
echo $GITHUB_TOKEN | docker login ghcr.io -u <username> --password-stdin
```

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
