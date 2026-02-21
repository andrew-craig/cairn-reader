# Fastlane Setup Guide for iOS TestFlight Deployment

This guide walks you through setting up Fastlane for automated iOS builds and TestFlight deployments.

## What You Just Got

✅ **Native iOS code** generated with `expo prebuild`
✅ **Fastlane** installed and configured
✅ **GitHub Actions workflow** ready for CI/CD
✅ **Build configuration** files ready to customize

## Prerequisites

1. **Apple Developer Account** - Active membership ($99/year)
2. **App Created in App Store Connect** with bundle ID: `com.cairnapp.cairnreader`
3. **GitHub Account** with this repository
4. **Xcode** installed on your Mac (for local testing)

---

## Step 1: Create App Store Connect API Key

This is the recommended authentication method (no 2FA issues).

### Create the API Key

1. Go to https://appstoreconnect.apple.com/access/api
2. Click the "+" button under "Keys"
3. Give it a name: "GitHub Actions CI/CD"
4. Set access level: **App Manager** (required for uploading builds)
5. Click **Generate**
6. **Download the .p8 file immediately** (you can only download once!)
7. Note down:
   - **Key ID** (e.g., `2X9R4HXF34`)
   - **Issuer ID** (UUID format, at top of page)

### Find Your Team ID

1. Go to https://developer.apple.com/account
2. Click "Membership" in the sidebar
3. Your **Team ID** is listed there (e.g., `A1B2C3D4E5`)

### Find Your App Store Connect App ID

1. Go to https://appstoreconnect.apple.com
2. Click on "My Apps"
3. Select your app (or create it if you haven't)
4. The App ID is in the URL: `https://appstoreconnect.apple.com/apps/{APP_ID}/`
5. Or find it under App Information → General Information → Apple ID

---

## Step 2: Set Up Fastlane Match (Certificate Management)

Fastlane Match stores your certificates and provisioning profiles in a private Git repository. This is the easiest way to manage code signing.

### Create a Private Certificates Repository

1. Go to GitHub (or GitLab, Bitbucket)
2. Create a **new private repository** named `certificates` (or `ios-certificates`)
3. **Do not** initialize it with a README
4. Copy the repository URL (e.g., `https://github.com/your-username/certificates`)

### Initialize Match

```bash
cd apps/mobile/ios
fastlane match init
```

When prompted:
- Select `git`
- Enter your certificates repository URL
- This will create/update the `Matchfile`

### Generate Certificates

**IMPORTANT:** This will revoke any existing certificates! If you're using existing certificates, you can import them instead (see Fastlane Match documentation).

```bash
# Set your Apple ID
export FASTLANE_APPLE_ID="your-apple-id@example.com"

# Generate App Store certificates
fastlane match appstore
```

You'll be prompted for:
1. Your Apple ID password (or app-specific password if 2FA is enabled)
2. A passphrase to encrypt the certificates (save this securely!)

This will:
- Create a distribution certificate
- Create an App Store provisioning profile
- Store them in your certificates repository
- Encrypt them with your passphrase

---

## Step 3: Set Up GitHub Secrets

Go to your GitHub repository → **Settings** → **Secrets and variables** → **Actions** → **New repository secret**

Add these secrets:

### Required Secrets

| Secret Name | Description | How to Get |
|-------------|-------------|------------|
| `APP_STORE_CONNECT_KEY` | Base64-encoded .p8 file | `base64 -i AuthKey_XXXXX.p8 \| pbcopy` |
| `APP_STORE_CONNECT_KEY_ID` | API Key ID | From App Store Connect (Step 1) |
| `APP_STORE_CONNECT_ISSUER_ID` | Issuer ID | From App Store Connect (Step 1) |
| `FASTLANE_APPLE_ID` | Your Apple ID email | your-apple-id@example.com |
| `MATCH_GIT_URL` | Certificates repo URL | `https://github.com/your-username/certificates` |
| `MATCH_PASSWORD` | Match encryption passphrase | The passphrase you set in Step 2 |
| `MATCH_GIT_BASIC_AUTHORIZATION` | GitHub auth token | See below |

### Creating MATCH_GIT_BASIC_AUTHORIZATION

This allows GitHub Actions to access your private certificates repository:

```bash
# Create a Personal Access Token (PAT) at:
# https://github.com/settings/tokens
#
# Required scopes:
# - repo (all)
#
# Then base64 encode it:
echo -n "your-username:your-PAT" | base64 | pbcopy
```

Paste the result as the `MATCH_GIT_BASIC_AUTHORIZATION` secret.

---

## Step 4: Update Configuration Files

### Update `apps/mobile/ios/fastlane/Matchfile`

The file is already configured to use environment variables, but verify these values:

```ruby
git_url(ENV["MATCH_GIT_URL"] || "https://github.com/your-org/certificates")
app_identifier(["com.cairnapp.cairnreader"])
username(ENV["FASTLANE_APPLE_ID"] || "your-apple-id@example.com")
```

### Update `apps/mobile/ios/ExportOptions.plist`

Replace `YOUR_TEAM_ID` with your actual Team ID:

```xml
<key>teamID</key>
<string>A1B2C3D4E5</string>
```

---

## Step 5: Test Locally (Recommended)

Before running in CI/CD, test the build locally:

```bash
cd apps/mobile/ios

# Set environment variables
export FASTLANE_APPLE_ID="your-apple-id@example.com"
export MATCH_GIT_URL="https://github.com/your-username/certificates"
export MATCH_PASSWORD="your-match-passphrase"
export APP_STORE_CONNECT_API_KEY_PATH="~/path/to/AuthKey_XXXXX.p8"

# Test build only (doesn't upload)
bundle exec fastlane build_only

# Or test full build + upload to TestFlight
bundle exec fastlane beta
```

If this works locally, it should work in GitHub Actions!

---

## Step 6: Trigger GitHub Actions

### Manual Trigger

1. Go to your repository on GitHub
2. Click **Actions** tab
3. Select **"iOS TestFlight (Fastlane)"** workflow
4. Click **"Run workflow"**
5. Select branch (usually `main`)
6. Click **"Run workflow"**

### Automatic Trigger on Tag

The workflow is configured to trigger automatically when you push a tag starting with `mobile-v`:

```bash
# Create and push a version tag
git tag mobile-v1.0.0
git push origin mobile-v1.0.0
```

---

## Understanding the Build Process

Here's what happens when you run the workflow:

1. **Checkout code** from GitHub
2. **Install Node.js** and npm dependencies
3. **Generate native iOS code** with `expo prebuild`
4. **Setup Ruby** and install Fastlane
5. **Install CocoaPods** dependencies
6. **Create App Store Connect API key file**
7. **Run Fastlane beta lane:**
   - Download certificates from Match
   - Increment build number
   - Build the app with Xcode
   - Upload IPA to TestFlight
8. **Cleanup** sensitive files

---

## Troubleshooting

### "Could not find a valid code signing identity"

- Make sure you ran `fastlane match appstore` successfully
- Verify your Team ID in `ExportOptions.plist`
- Check that certificates repository URL is correct

### "Authentication failed"

- Verify all GitHub secrets are set correctly
- Check that API Key hasn't expired
- Ensure API Key has "App Manager" or "Admin" role
- Verify `MATCH_GIT_BASIC_AUTHORIZATION` is correct

### "Could not find or download matching profile"

- Run `fastlane match appstore` again to regenerate profiles
- Check that bundle identifier matches: `com.cairnapp.cairnreader`
- Verify the app exists in App Store Connect

### "Build failed" with Xcode errors

- Check the GitHub Actions logs for specific errors
- Test the build locally first with `fastlane build_only`
- Ensure all native dependencies are properly linked

### "Match repository not found"

- Verify `MATCH_GIT_URL` secret is correct
- Check that `MATCH_GIT_BASIC_AUTHORIZATION` is valid
- Ensure the certificates repository exists and is private

### "Wrong passphrase for Match"

- Verify `MATCH_PASSWORD` secret matches what you set in Step 2
- Try re-running `fastlane match appstore` locally to verify

---

## Build Versioning

- **Version**: Controlled by `version` in `app.json` (e.g., `1.0.0`)
- **Build Number**: Auto-incremented by Fastlane on each build

To bump the version:

```bash
cd apps/mobile
# Edit app.json and change "version": "1.0.0" to "1.0.1"
# Then commit and push
```

---

## Fastlane Commands Reference

```bash
# All commands run from apps/mobile/ios/

# Test build without uploading
bundle exec fastlane build_only

# Build and upload to TestFlight
bundle exec fastlane beta

# Download/update certificates
bundle exec fastlane certificates

# See all available lanes
bundle exec fastlane lanes
```

---

## What Gets Committed to Git

✅ **Commit these:**
- `apps/mobile/ios/` (entire directory)
- `apps/mobile/ios/fastlane/` (Fastfile, Appfile, Matchfile)
- `.github/workflows/ios-testflight-fastlane.yml`
- `apps/mobile/app.json`
- `apps/mobile/eas.json`

❌ **Don't commit these** (already in .gitignore):
- `apps/mobile/ios/Pods/`
- `apps/mobile/ios/build/`
- Private keys (.p8 files)
- Certificates or provisioning profiles

---

## Keeping Your Setup Secure

🔒 **Security Best Practices:**

1. **Never commit** .p8 files or certificates to Git
2. **Use GitHub Secrets** for all sensitive data
3. **Enable 2FA** on your Apple ID
4. **Use app-specific passwords** if needed
5. **Restrict API Key** to only "App Manager" access (not Admin if not needed)
6. **Make certificates repo private**
7. **Rotate API Keys** periodically
8. **Review GitHub Actions logs** after each build

---

## Next Steps

1. ✅ Complete the setup steps above
2. ✅ Test a build locally
3. ✅ Trigger a GitHub Actions build
4. ✅ Check TestFlight for your build
5. ✅ Add internal testers in App Store Connect
6. 🎉 Share your app for testing!

---

## Additional Resources

- [Fastlane Documentation](https://docs.fastlane.tools/)
- [Fastlane Match Guide](https://docs.fastlane.tools/actions/match/)
- [App Store Connect API Keys](https://developer.apple.com/documentation/appstoreconnectapi/creating_api_keys_for_app_store_connect_api)
- [TestFlight Beta Testing](https://developer.apple.com/testflight/)
- [Expo Prebuild Guide](https://docs.expo.dev/workflow/prebuild/)

---

## Getting Help

If you run into issues:

1. Check the troubleshooting section above
2. Review GitHub Actions logs for errors
3. Test locally with `fastlane build_only` to isolate issues
4. Check Fastlane documentation for specific errors
5. Verify all secrets are set correctly in GitHub

---

**Happy Shipping! 🚀**
