#!/usr/bin/env bash
set -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
source "${REPO_ROOT}/scripts/k8s-utils.sh"
OPERATOR_DIR="${REPO_ROOT}/operator"
DEV_INTERNAL_DIR="${OPERATOR_DIR}/dev-internal/deploy" 

# Create the wavefront-operator yaml
operator_image_version="$(get_next_operator_version)${VERSION_POSTFIX}"
VERSION="${operator_image_version}" make -C "${OPERATOR_DIR}" released-kubernetes-yaml

# Replace the collector image version
collector_image_version="$(get_next_collector_version)${VERSION_POSTFIX}"
sed -i.bak "s%collector:.*$%collector: ${collector_image_version}%" "${DEV_INTERNAL_DIR}"/wavefront-operator.yaml
rm -f "${DEV_INTERNAL_DIR}"/wavefront-operator.yaml.bak

# Setup git config to push to remote repository
git config --global user.email "svc.wf-jenkins@vmware.com"
git config --global user.name "svc.wf-jenkins"
git remote set-url origin https://${TOKEN}@github.com/wavefronthq/observability-for-kubernetes.git

# Commit wavefront-operator.yaml to dev-internal/deploy directory
git add "${DEV_INTERNAL_DIR}"/wavefront-operator.yaml
echo "BRANCH_NAME: ${BRANCH_NAME}" || true
git commit -m "Build dev-internal/deploy/wavefront-operator.yaml from $(git rev-parse --short HEAD)" || exit 0
git push
git checkout .
