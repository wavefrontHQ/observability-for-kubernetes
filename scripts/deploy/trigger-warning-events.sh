#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

SCRIPT_DIR="$(dirname "$0")"

cd "$SCRIPT_DIR"

wait_for_namespace_created collector-targets

echo "Triggering warning events..."

kubectl apply -f pending-pod-image-cannot-be-loaded.yaml >/dev/null
