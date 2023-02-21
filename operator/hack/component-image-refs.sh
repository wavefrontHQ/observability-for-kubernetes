#!/bin/bash
set -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
OPERATOR_DIR="${REPO_ROOT}/operator"
cd "${OPERATOR_DIR}"
source "${REPO_ROOT}/scripts/k8s-utils.sh"

echo "kubernetes-collector:$(get_component_version collector)"
echo "kubernetes-operator-fluentbit:$(get_component_version logging)"
echo "proxy:$(get_component_version proxy)"
