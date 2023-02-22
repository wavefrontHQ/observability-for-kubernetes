#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

git show origin/rc:operator/wavefront-operator-main.yaml > ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml
OPERATOR_ALPHA_TAG=$(cat ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml | yq 'select(.metadata.name == "wavefront-controller-manager" and .kind == "Deployment" ) | .spec.template.spec.containers[0].image' | cut -d ':' -f2)
COLLECTOR_ALPHA_TAG=$(cat ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml | yq 'select(.metadata.name == "wavefront-component-versions" ) | .data.collector')
OPERATOR_VERSION=$(get_operator_version)
COLLECTOR_VERSION=$(get_component_version collector)
crane copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-operator:${OPERATOR_ALPHA_TAG}" "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-operator-snapshot:${OPERATOR_VERSION}"
crane copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-collector:${COLLECTOR_ALPHA_TAG}" "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-collector-snapshot:${COLLECTOR_VERSION}"
