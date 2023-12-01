#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

SCRIPT_DIR="$(dirname "$0")"
cd "$SCRIPT_DIR"

echo "Deploying k8s event targets..."

kubectl patch -n collector-targets pod/pod-stuck-in-terminating -p '{"metadata":{"finalizers":null}}' &>/dev/null || true
kubectl delete --ignore-not-found=true namespace collector-targets &>/dev/null || true

wait_for_namespace_created collector-targets
wait_for_namespaced_resource_created collector-targets serviceaccount/default

kubectl apply -f pending-pod-image-cannot-be-loaded.yaml >/dev/null
kubectl apply -f pending-pod-cannot-be-scheduled.yaml >/dev/null

echo "Finished deploying k8s event targets"
