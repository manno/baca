#!/bin/bash
#
# SCRIPT: build-multiarch.sh
# DESCRIPTION: Builds a multi-architecture (linux/amd64, linux/arm64) Docker image
#              and pushes it to GitHub Container Registry (GHCR).
#
# PREREQUISITES:
# 1. Docker and the Buildx plugin must be installed.
# 2. You must have a Personal Access Token (PAT) with 'write:packages' scope.
# 3. The Dockerfile must be in the current directory.

IMAGE_NAME="ghcr.io/manno/background-coder"
TAG="latest"
PLATFORMS=${PLATFORMS-"linux/amd64,linux/arm64"}
DOCKERFILE="Dockerfile"

# Function to perform the build and push
FULL_IMAGE_TAG="${IMAGE_NAME}:${TAG}"

echo "Starting multi-arch build for platforms: ${PLATFORMS}"
echo "Targeting image: ${FULL_IMAGE_TAG}"

# The core buildx command:
# --push sends the image directly to the registry
# --platform specifies the target architectures
docker buildx build \
    --builder buildx-multi-arch \
    --platform "${PLATFORMS}" \
    -f "${DOCKERFILE}" \
    -t "${FULL_IMAGE_TAG}" \
    --load .

if [ $? -eq 0 ]; then
    echo "--------------------------------------------------------"
    echo "✅ SUCCESS! Multi-arch image built."
    echo "Image: ${FULL_IMAGE_TAG}"
    echo "--------------------------------------------------------"
else
    echo "--------------------------------------------------------"
    echo "❌ FAILURE! Docker Buildx failed."
    echo "--------------------------------------------------------"
    exit 1
fi
