#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)

RELEASE=$("$REPO_ROOT"/bin/helm list -n wavefront | awk '/wavefront/ {print $1}')
if [ -z ${RELEASE} ]; then exit 0; fi

echo "uninstalling wavefront helm release"
helm uninstall wavefront --namespace wavefront &>/dev/null || true
kubectl delete namespace wavefront &>/dev/null || true