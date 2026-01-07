#!/bin/bash

# GitHub Self-Hosted Runner Setup Script for PhotoSync
# This script sets up a GitHub Actions runner on this Mac for iOS and Android builds

set -e

REPO="BenJaminBMorin/photosync"
RUNNER_NAME="photosync-mac-runner"
RUNNER_LABELS="self-hosted,macos,photosync"
RUNNER_DIR="$HOME/actions-runner"

echo "=========================================="
echo "GitHub Actions Self-Hosted Runner Setup"
echo "Repository: $REPO"
echo "Runner Name: $RUNNER_NAME"
echo "=========================================="
echo ""

# Check if runner directory already exists
if [ -d "$RUNNER_DIR" ]; then
    echo "⚠️  Runner directory already exists at $RUNNER_DIR"
    read -p "Do you want to remove it and start fresh? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "Removing existing runner directory..."
        rm -rf "$RUNNER_DIR"
    else
        echo "Exiting. Please manually remove or configure the existing runner."
        exit 1
    fi
fi

# Create runner directory
echo "Creating runner directory at $RUNNER_DIR..."
mkdir -p "$RUNNER_DIR"
cd "$RUNNER_DIR"

# Download the latest runner package for macOS
echo "Downloading GitHub Actions runner..."
RUNNER_VERSION="2.321.0"
ARCH=$(uname -m)

if [ "$ARCH" = "arm64" ]; then
    RUNNER_ARCH="arm64"
else
    RUNNER_ARCH="x64"
fi

RUNNER_FILE="actions-runner-osx-${RUNNER_ARCH}-${RUNNER_VERSION}.tar.gz"
curl -o "$RUNNER_FILE" -L "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/${RUNNER_FILE}"

# Extract the installer
echo "Extracting runner package..."
tar xzf "$RUNNER_FILE"
rm "$RUNNER_FILE"

echo ""
echo "=========================================="
echo "NEXT STEPS:"
echo "=========================================="
echo ""
echo "1. Get a runner registration token:"
echo "   Run: gh api repos/$REPO/actions/runners/registration-token --jq .token"
echo ""
echo "2. Configure the runner with the token:"
echo "   cd $RUNNER_DIR"
echo "   ./config.sh --url https://github.com/$REPO --token YOUR_TOKEN_HERE --name $RUNNER_NAME --labels $RUNNER_LABELS"
echo ""
echo "3. Install and start the runner as a service (runs in background):"
echo "   cd $RUNNER_DIR"
echo "   ./svc.sh install"
echo "   ./svc.sh start"
echo ""
echo "4. Verify the runner is running:"
echo "   ./svc.sh status"
echo ""
echo "=========================================="
echo ""
echo "The runner will:"
echo "- Run as a macOS LaunchAgent"
echo "- Start automatically on login"
echo "- Run in the background"
echo ""
echo "To stop the runner later: cd $RUNNER_DIR && ./svc.sh stop"
echo "To uninstall: cd $RUNNER_DIR && ./svc.sh uninstall"
echo ""
