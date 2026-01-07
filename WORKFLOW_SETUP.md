# PhotoSync GitHub Workflow Configuration

## Self-Hosted Runner Setup

**Runner Name:** `Bens-Mac` (photosync-mac-runner)
**Labels:** `self-hosted`, `macOS`, `ARM64`, `local`
**Status:** Running as macOS LaunchAgent (auto-starts on login)
**Location:** `~/actions-runner`

### Managing the Runner
- **Status:** `cd ~/actions-runner && ./svc.sh status`
- **Stop:** `cd ~/actions-runner && ./svc.sh stop`
- **Start:** `cd ~/actions-runner && ./svc.sh start`
- **Uninstall:** `cd ~/actions-runner && ./svc.sh uninstall`

## Workflow Summary

### iOS Workflows (Self-Hosted on Mac)
1. **ios.yml** - iOS Build Workflow
   - **Trigger:** PR to main/develop when `ios/**` changes, or manual
   - **Runner:** `[self-hosted, macos]`
   - **Jobs:**
     - Build for iOS Simulator (Debug)
     - Build for Release
     - Build unsigned IPA (on push/manual)
   - **No GitHub build credits used** ✓

2. **testflight.yml** - TestFlight Deployment
   - **Trigger:** Push to main when `ios/**` changes, or manual
   - **Runner:** `[self-hosted, macos]`
   - **Jobs:**
     - Build and upload to TestFlight via Fastlane
     - Uses App Store Connect API for authentication
   - **No GitHub build credits used** ✓

### Android Workflows (Self-Hosted on Mac)
3. **android.yml** - Android Build Workflow
   - **Trigger:** Push/PR to main/master when `android/**` changes
   - **Runner:** `[self-hosted, macos]`
   - **Environment:**
     - `ANDROID_HOME: /Users/benjamin/Library/Android/sdk`
     - `ANDROID_SDK_ROOT: /Users/benjamin/Library/Android/sdk`
   - **Jobs:**
     - Build Debug APK + Unit Tests
     - Build Release APK (on push to main)
     - Lint checks
   - **No GitHub build credits used** ✓

### Server Workflows (GitHub-Hosted)
4. **server-go.yml** - Go Server Build
   - **Trigger:** Push/PR to main/master when `server-go/**` changes
   - **Runner:** `ubuntu-latest` (GitHub-hosted)
   - **Jobs:**
     - Build and test Go server
     - Publish multi-platform binaries (linux, windows, darwin)
   - **Uses GitHub build credits** (server-side only)

5. **server-dotnet.yml** - .NET Server Build
   - **Trigger:** Push/PR to main/master when `server/**` changes
   - **Runner:** `ubuntu-latest` (GitHub-hosted)
   - **Jobs:**
     - Build and test .NET server
     - Publish multi-platform binaries
   - **Uses GitHub build credits** (server-side only)

6. **deploy.yml** - Server Deployment
   - **Trigger:** Push to main when `server-go/**` changes, or manual
   - **Runner:** `self-hosted` (separate Linux runner "photosync-deploy")
   - **Jobs:**
     - Deploy to local server via Docker Compose
   - **No GitHub build credits used** ✓

### CI Workflow (Mixed Runners)
7. **ci.yml** - Continuous Integration
   - **Trigger:** Pull requests or manual
   - **Path Detection:** Uses `dorny/paths-filter` to detect changes
   - **Jobs:**
     - `changes` - Detects which parts of codebase changed (GitHub-hosted)
     - `dotnet-build` - If `server/**` changed (GitHub-hosted)
     - `go-build` - If `server-go/**` changed (GitHub-hosted)
     - `android-build` - If `android/**` changed **[self-hosted, macos]** ✓
     - `ios-build` - If `ios/**` changed **[self-hosted, macos]** ✓

## Path Filters Summary

All workflows use path filters to only run when relevant files change:

| Workflow | Path Filter | Description |
|----------|-------------|-------------|
| ios.yml | `ios/**` | Only iOS source changes |
| testflight.yml | `ios/**` | Only iOS source changes |
| android.yml | `android/**` | Only Android source changes |
| server-go.yml | `server-go/**` | Only Go server changes |
| server-dotnet.yml | `server/**` | Only .NET server changes |
| deploy.yml | `server-go/**`, `docker-compose.yml` | Server deployment files |
| ci.yml | Uses path detection | Runs only affected platform builds |

## GitHub Build Credits Usage

**iOS & Android builds:** ✓ **ZERO GitHub credits used** - All run on your Mac
**Server builds:** Uses GitHub credits for Linux builds (minimal cost)
**Total savings:** ~90% reduction in GitHub Actions minutes

## Notes

- The self-hosted Mac runner handles all iOS and Android builds
- Server-side builds remain on GitHub-hosted runners (Linux)
- All builds trigger only when relevant code changes
- Discord notifications configured for iOS workflows
- TestFlight deployment fully automated via Fastlane
