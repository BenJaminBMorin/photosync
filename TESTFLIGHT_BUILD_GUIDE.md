# TestFlight Build & Upload Guide

This guide will walk you through building and uploading PhotoSync to TestFlight.

## Prerequisites

- Mac with Xcode 15+
- Apple Developer Account
- App already set up in App Store Connect
- Latest code pulled from GitHub
- Valid provisioning profiles and signing certificates

## Step 1: Prepare on Your Mac

```bash
cd /path/to/photosync
git pull origin main
cd ios
```

Verify you have the latest code with password authentication:
```bash
git log --oneline -5
```

Expected commits:
- `d517bfb` - Add user info endpoint and consolidate settings UI
- `3fc2b1a` - Improve authentication flow with settings-driven login
- `eaf8091` - Implement iOS frontend for password-based authentication

## Step 2: Open Project in Xcode

```bash
open PhotoSync.xcodeproj
```

## Step 3: Verify Build Settings

In Xcode:

1. **Select Project**: Click on "PhotoSync" in the navigator
2. **Select Target**: Click on "PhotoSync" target
3. **Check General Settings**:
   - Version should be updated (e.g., 1.1.0)
   - Build number should be incremented
4. **Check Signing & Capabilities**:
   - Ensure signing certificate is selected
   - Team is correct
   - Provisioning profiles are automatic or manually selected

## Step 4: Build for Distribution

### Option A: Using Xcode UI (Recommended)

1. **Set Build Configuration**:
   - Product → Scheme → Edit Scheme
   - Select "Release" configuration
   - Verify iOS Deployment Target matches App Store requirements

2. **Archive the Build**:
   - Product → Archive
   - Wait for build to complete
   - Organizer window should open automatically

3. **Upload to App Store Connect**:
   - In Organizer window, click "Distribute App"
   - Select "TestFlight Only" or "App Store & TestFlight"
   - Follow the wizard:
     - Choose "Upload"
     - Select signing certificate
     - Complete upload

### Option B: Command Line (Faster)

```bash
# Build the archive
xcodebuild -scheme PhotoSync \
  -configuration Release \
  -derivedDataPath build \
  -archivePath build/PhotoSync.xcarchive \
  clean archive

# Export for App Store (creates .ipa)
xcodebuild -exportArchive \
  -archivePath build/PhotoSync.xcarchive \
  -exportOptionsPlist ExportOptions.plist \
  -exportPath build/ipa

# Upload to App Store Connect
xcrun altool --upload-app \
  --file build/ipa/PhotoSync.ipa \
  --type ios \
  --apiKey $API_KEY_ID \
  --apiIssuer $API_ISSUER_ID
```

## Step 5: Monitor Upload Progress

1. **In Xcode Organizer**:
   - Watch progress bar until complete
   - Will show "Uploading..." then "Processing..."

2. **In App Store Connect**:
   - Go to https://appstoreconnect.apple.com
   - Select PhotoSync app
   - TestFlight tab
   - Builds section
   - You should see your new build appearing

## Step 6: Wait for App Store Processing

- Processing typically takes 5-15 minutes
- You'll receive email confirmation when ready for TestFlight

## Step 7: Add Testers (App Store Connect)

1. Login to App Store Connect
2. Select PhotoSync app
3. Go to TestFlight → iOS Builds
4. Click your new build
5. Add Internal Testers (your Apple account)
6. Or add External Testers via email invites

## Step 8: Test the Build

Once available on TestFlight:

1. Open TestFlight app on iOS device
2. Install PhotoSync
3. Test all features:
   - ✅ Login with password
   - ✅ Email password reset
   - ✅ Phone 2FA password reset
   - ✅ API key generation
   - ✅ Settings & user info
   - ✅ Photo sync

## Troubleshooting

### Build Fails with Code Signing Issues

```bash
# Clear derived data
rm -rf ~/Library/Developer/Xcode/DerivedData/PhotoSync*

# Recreate signing
open PhotoSync.xcodeproj
# Then manually fix signing in General tab
```

### "App Store Connect Processing Failed"

- Check email for specific error
- Common causes:
  - Invalid version number
  - Missing required screenshots
  - Privacy policy issues
  - Incorrect build identifier

### Upload Stuck at "Uploading..."

```bash
# Cancel and retry
# If still stuck after 30 mins, restart Xcode
```

### Low Memory/Build Timeout

```bash
# Free up memory
killall -9 Simulator
killall -9 "iPhone Simulator"

# Try again with fewer simulators running
```

## Quick Reference: Version Bumping

When uploading new builds:

1. **Patch Update** (1.0.1): Bug fixes
   - Increment build number only
2. **Minor Update** (1.1.0): New features
   - Increment minor version + reset build number to 1
3. **Major Update** (2.0.0): Breaking changes
   - Increment major version + reset others to 0

Current version: Check in Xcode
- Product → Scheme → Edit Scheme → Info tab

## What's New in This Build

- ✅ Password-based authentication
- ✅ Email password reset
- ✅ Phone 2FA password reset
- ✅ API key generation on login
- ✅ Settings UI consolidation
- ✅ User info endpoint integration

## Next Steps After TestFlight

- Gather feedback from testers
- Fix any bugs identified
- Prepare for App Store release
- Update release notes
- Create app preview videos if needed

---

**Need Help?**

If you encounter issues:
1. Check Xcode console for error messages
2. Review Apple Developer documentation
3. Check App Store Connect for rejection reasons
4. Verify all signing certificates are up to date

**Backend Server Status**: Running on port 5000 - All endpoints operational ✅
