#!/bin/bash
set -euxo pipefail

# The upstream cluster to import all the images to.
upstream_ctx="${CTX-k3d-upstream}"

k3d image import ghcr.io/manno/background-coder:latest -c "${upstream_ctx#k3d-}"
