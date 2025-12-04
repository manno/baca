#!/usr/bin/env bash
# Run integration tests with ginkgo

set -e

if [ -z "$KUBEBUILDER_ASSETS" ]; then
  echo "Setting KUBEBUILDER_ASSETS..."
  export KUBEBUILDER_ASSETS=$(setup-envtest use -p path latest)
fi

echo "Using KUBEBUILDER_ASSETS: $KUBEBUILDER_ASSETS"
echo ""

exec ginkgo -v "$@" ./tests/...
