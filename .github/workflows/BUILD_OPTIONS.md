# iOS Build & Deployment Options Comparison

This document compares different approaches for building and deploying your Expo/React Native app to TestFlight.

## Quick Comparison

| Approach | Cost | Setup Complexity | Control | CI/CD Time | Best For |
|----------|------|------------------|---------|------------|----------|
| **EAS Build** | $29-299/mo* | ⭐ Easy | Medium | ~15-20 min | Quick setup, small teams |
| **Fastlane** | Free | ⭐⭐⭐ Complex | High | ~25-30 min | Production apps, teams |
| **Xcode Cloud** | Free-$15/mo** | ⭐⭐ Medium | Medium | ~20-25 min | Apple ecosystem focus |
| **Local Build** | Free | ⭐⭐ Medium | High | ~30-40 min | Full control needed |

\* EAS Build pricing: Free tier available, paid plans for private apps and priority builds
\** Xcode Cloud: 25 hours/month free, then $15/mo for 100 hours

---

## Option 1: EAS Build (What We Set Up First)

**Files:** `.github/workflows/ios-testflight.yml`, `eas.json`

### Pros
- ✅ Easiest setup - minimal configuration
- ✅ No need to manage certificates/provisioning profiles
- ✅ Automatic build number incrementation
- ✅ Built-in OTA updates support
- ✅ Great for managed Expo workflow
- ✅ Excellent documentation and community support

### Cons
- ❌ Costs money for private apps (after free tier)
- ❌ Less control over build environment
- ❌ Dependent on Expo's infrastructure
- ❌ Requires Expo account

### Cost
- **Free tier**: 30 builds/month for personal accounts
- **Production**: $29/month (unlimited builds, priority queue)
- **Enterprise**: $299/month (advanced features, dedicated support)

### When to Use
- You're okay with the cost
- You want the easiest setup and maintenance
- You're using other Expo services
- You don't need custom native code modifications

### Setup Steps
1. Create Expo account
2. Set GitHub secrets (EXPO_TOKEN, etc.)
3. Run workflow

**Status:** ✅ Already configured in your repo

---

## Option 2: Fastlane (Industry Standard)

**Files:** `.github/workflows/ios-testflight-fastlane.yml`, `fastlane/Fastfile`

### Pros
- ✅ Completely free and open source
- ✅ Industry standard for iOS deployment
- ✅ Highly customizable
- ✅ Works with any React Native/Expo app
- ✅ Great for complex workflows
- ✅ Certificate management with Fastlane Match

### Cons
- ❌ More complex setup
- ❌ Need to manage certificates/provisioning profiles
- ❌ Requires running `expo prebuild` (generates native code)
- ❌ Longer CI/CD run times (needs macOS runner)
- ❌ More GitHub Actions minutes used

### Cost
- **Fastlane:** Free
- **GitHub Actions:** Uses macOS runners (10x Linux cost)
  - GitHub Free: 0 macOS minutes included
  - GitHub Pro: 0 macOS minutes included
  - GitHub Team/Enterprise: Pay per minute or included in plan

### When to Use
- You want full control over the build process
- You need custom native code
- Cost savings vs EAS (if you have GitHub Actions minutes)
- You're building a production app
- You want to learn industry-standard tools

### Setup Steps
1. Run `npx expo prebuild` to generate native code
2. Set up Fastlane Match for certificate management
3. Configure GitHub secrets
4. Update Matchfile with your certificate repo
5. Run workflow

**Status:** ✅ Template files created

---

## Option 3: Xcode Cloud (Apple's Solution)

### Pros
- ✅ Native Apple solution
- ✅ Integrated with Xcode and App Store Connect
- ✅ Generous free tier (25 hours/month)
- ✅ Automatic certificate management
- ✅ Built-in TestFlight integration

### Cons
- ❌ Apple ecosystem lock-in
- ❌ Less flexible than Fastlane
- ❌ Requires Xcode project (need `expo prebuild`)
- ❌ Learning curve if you haven't used it

### Cost
- **Free tier:** 25 compute hours/month
- **Paid plans:** $15/mo for 100 hours, $150/mo for 1000 hours

### When to Use
- You prefer Apple's ecosystem
- You want automatic certificate management
- You're building only iOS (or iOS+macOS)
- You want tight App Store Connect integration

### Setup Steps
1. Run `npx expo prebuild` to generate native code
2. Open project in Xcode
3. Enable Xcode Cloud in Xcode settings
4. Configure workflow in Xcode Cloud
5. No GitHub Actions needed (uses Apple's infrastructure)

**Status:** ❌ Not configured (requires Xcode setup)

---

## Option 4: Manual Local Builds

**Files:** `.github/workflows/ios-testflight-local-build.yml`

### Pros
- ✅ Free (uses GitHub Actions)
- ✅ Full control over build process
- ✅ No third-party dependencies (besides GitHub)
- ✅ Can customize every step

### Cons
- ❌ Most complex setup
- ❌ Manual certificate/provisioning profile management
- ❌ Longest CI/CD times
- ❌ More maintenance overhead
- ❌ Most expensive in GitHub Actions minutes (macOS runners)

### Cost
- **Free** if you have GitHub Actions minutes
- **macOS runners** are 10x cost of Linux runners

### When to Use
- You need absolute control
- You have specific build requirements
- You're learning iOS deployment
- You want zero dependency on third-party services

### Setup Steps
1. Run `npx expo prebuild` to generate native code
2. Export certificates and provisioning profiles
3. Set up all GitHub secrets
4. Test build locally first
5. Run workflow

**Status:** ⚠️ Template created (needs certificates setup)

---

## Recommendation by Use Case

### Just Getting Started
**Use: EAS Build** - It's the fastest way to get to TestFlight. Pay the $29/month and focus on building your app.

### Serious About This App / Small Budget
**Use: Fastlane** - One-time setup cost (time), then free forever. Industry standard, great for production.

### Already Using Apple Ecosystem
**Use: Xcode Cloud** - If you're comfortable with Xcode and want Apple's integrated solution.

### Need Maximum Control / Have DevOps Experience
**Use: Manual Local Builds or Fastlane** - Full control, more complexity.

---

## Migration Path

You can start with one approach and migrate to another:

1. **Start with EAS** → Great for MVP/initial development
2. **Move to Fastlane** → When you want to reduce costs or need more control
3. **Stay with EAS if** → The cost is negligible compared to development time saved

**Note:** Once you run `expo prebuild`, you commit to managing native code. You can still use EAS, but you're no longer in "managed workflow."

---

## Current Setup in This Repo

✅ **EAS Build** - Fully configured and ready to use
✅ **Fastlane** - Template files created, needs certificate setup
✅ **Local Build** - Template created, needs certificate setup

**Recommended Next Step:**
1. Try EAS Build first (easiest, fastest)
2. If cost becomes an issue or you need more control, switch to Fastlane
3. Keep the templates for reference

---

## Additional Resources

- [EAS Build Pricing](https://expo.dev/pricing)
- [Fastlane Documentation](https://docs.fastlane.tools/)
- [Xcode Cloud Overview](https://developer.apple.com/xcode-cloud/)
- [GitHub Actions Pricing](https://docs.github.com/en/billing/managing-billing-for-github-actions/about-billing-for-github-actions)
- [Expo Prebuild Guide](https://docs.expo.dev/workflow/prebuild/)
