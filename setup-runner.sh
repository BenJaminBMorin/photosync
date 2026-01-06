#!/bin/bash
set -e

echo "Setting up GitHub Actions self-hosted runner..."

# Create a folder for the runner
mkdir -p ~/actions-runner && cd ~/actions-runner

# Download the latest runner package
echo "Downloading GitHub Actions runner..."
curl -o actions-runner-osx-arm64-2.321.0.tar.gz -L https://github.com/actions/runner/releases/download/v2.321.0/actions-runner-osx-arm64-2.321.0.tar.gz

# Extract the installer
echo "Extracting runner..."
tar xzf ./actions-runner-osx-arm64-2.321.0.tar.gz

echo ""
echo "========================================"
echo "Next steps:"
echo "1. Go to: https://github.com/BenJaminBMorin/photosync/settings/actions/runners/new"
echo "2. Copy the registration token"
echo "3. Run this command:"
echo "   cd ~/actions-runner"
echo "   ./config.sh --url https://github.com/BenJaminBMorin/photosync --token YOUR_TOKEN --labels macos,local"
echo "4. Then run: ./run.sh"
echo "========================================"
