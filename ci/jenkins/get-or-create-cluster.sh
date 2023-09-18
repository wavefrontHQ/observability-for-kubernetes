#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

# Explicitly expect GKE_CLUSTER_NAME to be set in env vars
make -C "${REPO_ROOT}" gke-cluster-name-check

# Assume it's because it doesn't exist and create it
if ! make -C "${REPO_ROOT}" gke-connect-to-cluster; then
    echo "Did not find cluster '${GKE_CLUSTER_NAME}', creating ..."
    make -C "${REPO_ROOT}" create-gke-cluster GKE_EXPIRES_IN_HOURS=10
    exit 0
fi

echo "Found and connected to cluster '${GKE_CLUSTER_NAME}'"
