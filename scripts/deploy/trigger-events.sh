#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

SCRIPT_DIR="$(dirname "$0")"

cd "$SCRIPT_DIR"

wait_for_namespace_created collector-targets

echo "Triggering events..."

kubectl apply -f prom-example.yaml >/dev/null

wait_for_cluster_ready "collector-targets"

sed -e "s/TIMESTAMP/$(date -u +"%Y-%m-%dT%H:%M:%SZ")/g" warning-event.yaml | kubectl apply -f -
