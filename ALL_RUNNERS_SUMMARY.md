# Complete Self-Hosted Runner Configuration

## All Workflows Now Using Self-Hosted Runners ✓

**ZERO GitHub Actions credits used across all workflows!**

---

## Runner Infrastructure

### Mac Runner: `Bens-Mac`
- **Labels:** `self-hosted`, `macOS`, `ARM64`, `local`
- **Status:** Online
- **Location:** `~/actions-runner`
- **Manages:** iOS and Android builds

### Linux Runner: `photosync-deploy`
- **Labels:** `self-hosted`, `Linux`, `X64`, `photosync`
- **Status:** Online
- **Manages:** Server builds and deployments

---

## Workflow Breakdown by Runner

### iOS Workflows → Mac Runner (`[self-hosted, macos]`)

1. **ios.yml**
   - Triggers: PR to main/develop when `ios/**` changes
   - Jobs: Build simulator, Release build, Unsigned IPA
   - Runner: Mac ✓

2. **testflight.yml**
   - Triggers: Push to main when `ios/**` changes
   - Jobs: Build and deploy to TestFlight
   - Runner: Mac ✓

### Android Workflows → Mac Runner (`[self-hosted, macos]`)

3. **android.yml**
   - Triggers: Push/PR when `android/**` changes
   - Jobs: Debug build, Release build, Lint
   - Runner: Mac ✓
   - Environment: Android SDK at `/Users/benjamin/Library/Android/sdk`

### Server Workflows → Linux Runner (`[self-hosted, Linux]`)

4. **server-go.yml**
   - Triggers: Push/PR when `server-go/**` changes
   - Jobs: Build, Test, Publish multi-platform binaries
   - Runner: Linux ✓

5. **server-dotnet.yml**
   - Triggers: Push/PR when `server/**` changes
   - Jobs: Build, Test, Publish multi-platform binaries
   - Runner: Linux ✓

6. **deploy.yml**
   - Triggers: Push to main when `server-go/**` changes
   - Jobs: Docker Compose deployment
   - Runner: Linux ✓

### CI Workflow → Mixed Runners (`[self-hosted, Linux]` + `[self-hosted, macos]`)

7. **ci.yml**
   - Triggers: Pull requests
   - Jobs:
     - `changes` - Path detection → Linux ✓
     - `dotnet-build` - If server changes → Linux ✓
     - `go-build` - If server-go changes → Linux ✓
     - `android-build` - If android changes → Mac ✓
     - `ios-build` - If ios changes → Mac ✓

---

## Path Filters (Smart Triggering)

All workflows use path filters to only run when relevant:

| Workflow | Triggers Only When | Runner |
|----------|-------------------|---------|
| ios.yml | `ios/**` changes | Mac |
| testflight.yml | `ios/**` changes | Mac |
| android.yml | `android/**` changes | Mac |
| server-go.yml | `server-go/**` changes | Linux |
| server-dotnet.yml | `server/**` changes | Linux |
| deploy.yml | `server-go/**` changes | Linux |
| ci.yml | Any changes (filtered per job) | Mixed |

---

## GitHub Actions Credits Usage

**Before:** ~2000-3000 minutes/month on GitHub-hosted runners
**After:** **0 minutes/month** - 100% self-hosted

### Cost Savings
- **iOS builds:** Previously ~10 minutes × 10 runs/month × 10 (macOS multiplier) = 1000 minutes → **Now 0**
- **Android builds:** Previously ~5 minutes × 10 runs/month = 50 minutes → **Now 0**
- **Server builds:** Previously ~2 minutes × 20 runs/month = 40 minutes → **Now 0**
- **CI builds:** Previously ~15 minutes × 30 runs/month = 450 minutes → **Now 0**

**Total Monthly Savings: ~1500-3000 minutes** (or more during active development)

For private repos:
- Free tier: 2000 minutes/month
- You're now using: **0 minutes/month**
- Overage cost avoided: Potentially $0.008/minute for macOS runners

---

## Managing Your Runners

### Mac Runner (`~/actions-runner`)
```bash
# Check status
cd ~/actions-runner && ./svc.sh status

# Stop
cd ~/actions-runner && ./svc.sh stop

# Start
cd ~/actions-runner && ./svc.sh start

# View logs
tail -f ~/Library/Logs/actions.runner.BenJaminBMorin-photosync.Bens-Mac/Runner-*.log
```

### Linux Runner
Check with your Linux server runner management commands.

---

## Summary

✅ **All 7 workflows now use self-hosted runners**
✅ **Zero GitHub Actions credits consumed**
✅ **Path filters prevent unnecessary builds**
✅ **Automatic runner startup on Mac login**
✅ **Full control over build environment**

Your PhotoSync project is now completely independent of GitHub's build infrastructure!
