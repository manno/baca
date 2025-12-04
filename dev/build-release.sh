#!/bin/bash
#
# SCRIPT: build-release.sh
# DESCRIPTION: Builds release binary for specified platform
#
# Usage:
#   ./dev/build-release.sh                    # Builds for linux/amd64 (default)
#   GOARCH=arm64 ./dev/build-release.sh       # Builds for linux/arm64
#   GOOS=darwin GOARCH=arm64 ./dev/build-release.sh  # Builds for macOS arm64

set -e

# Set defaults
export GOOS="${GOOS:-linux}"
export GOARCH="${GOARCH:-amd64}"

# Create dist directory if it doesn't exist
mkdir -p dist

echo "Building baca for $GOOS/$GOARCH..."

CGO_ENABLED=0 go build -gcflags='all=-N -l' -o "dist/baca-$GOOS-$GOARCH" .

echo "✅ Binary built: dist/baca-$GOOS-$GOARCH"
