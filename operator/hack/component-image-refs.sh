#!/bin/bash
set -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
OPERATOR_DIR="${REPO_ROOT}/operator"
cd "${OPERATOR_DIR}"

echo "kubernetes-collector:$(yq .data.collector "${OPERATOR_DIR}"/config/manager/component_versions.yaml)"
echo "kubernetes-operator-fluentbit:$(yq .data.logging "${OPERATOR_DIR}"/config/manager/component_versions.yaml)"
echo "proxy:$(yq .data.proxy "${OPERATOR_DIR}"/config/manager/component_versions.yaml)"
