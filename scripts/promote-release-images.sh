#!/bin/bash
set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

# Get next release versions
OPERATOR_VERSION=$(get_next_operator_version)
COLLECTOR_VERSION=$(get_next_collector_version)

# Copy last tested wavefront-operator yaml from rc branch to dev-internal
git show origin/rc:operator/wavefront-operator-main.yaml > ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml

# Promote alpha images to release versions
OPERATOR_ALPHA_IMAGE=$(cat "${REPO_ROOT}"/operator/dev-internal/deploy/wavefront-operator.yaml | yq 'select(.metadata.name == "wavefront-controller-manager" and .kind == "Deployment" ) | .spec.template.spec.containers[0].image')
OPERATOR_ALPHA_TAG=$(echo ${OPERATOR_ALPHA_IMAGE} | cut -d ':' -f2)
COLLECTOR_ALPHA_TAG=$(cat "${REPO_ROOT}"/operator/dev-internal/deploy/wavefront-operator.yaml | yq 'select(.metadata.name == "wavefront-component-versions" ) | .data.collector')
crane copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-operator:${OPERATOR_ALPHA_TAG}" "projects.registry.vmware.com/tanzu_observability/kubernetes-operator:${OPERATOR_VERSION}" || true
crane validate --remote "projects.registry.vmware.com/tanzu_observability/kubernetes-operator:${OPERATOR_VERSION}"
crane copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-collector:${COLLECTOR_ALPHA_TAG}" "projects.registry.vmware.com/tanzu_observability/kubernetes-collector:${COLLECTOR_VERSION}" || true
crane validate --remote "projects.registry.vmware.com/tanzu_observability/kubernetes-collector:${COLLECTOR_VERSION}"

# Update wavefront-operator yaml in dev-internal with release versions
sed -i.bak "s/collector: ${COLLECTOR_ALPHA_TAG}/collector: ${COLLECTOR_VERSION}/g" ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml
sed -i.bak "s#image: ${OPERATOR_ALPHA_IMAGE}#image: projects.registry.vmware.com/tanzu_observability/kubernetes-operator:${OPERATOR_VERSION}#g" ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml
rm -rf ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml.bak

# Update custom-configuration md in dev-internal with release versions
sed -i.bak "s%kubernetes-collector:[0-9.]*%kubernetes-collector:${COLLECTOR_VERSION}%g" ${REPO_ROOT}/operator/dev-internal/docs/operator/custom-configuration.md
sed -i.bak "s%kubernetes-operator:[0-9.]*%kubernetes-operator:${OPERATOR_VERSION}%g" ${REPO_ROOT}/operator/dev-internal/docs/operator/custom-configuration.md
rm -rf ${REPO_ROOT}/operator/dev-internal/docs/operator/custom-configuration.md.bak

# Promote dev-internal to top level
pushd ${REPO_ROOT}
  make promote-internal
popd

# Update release versions
yq -i e ".data.collector |= \"$COLLECTOR_VERSION\"" "${REPO_ROOT}"/operator/config/manager/component_versions.yaml
echo "$COLLECTOR_VERSION" >"${REPO_ROOT}"/collector/release/VERSION
echo "$OPERATOR_VERSION" >"${REPO_ROOT}"/operator/release/OPERATOR_VERSION
NEXT_OPERATOR_RELEASE_VERSION="$("${REPO_ROOT}"/scripts/get-bumped-version.sh -v "${OPERATOR_VERSION}" -s minor)"
echo "$NEXT_OPERATOR_RELEASE_VERSION" >"${REPO_ROOT}"/operator/release/NEXT_RELEASE_VERSION
NEXT_COLLECTOR_RELEASE_VERSION="$("${REPO_ROOT}"/scripts/get-bumped-version.sh -v "${COLLECTOR_VERSION}" -s minor)"
echo "$NEXT_COLLECTOR_RELEASE_VERSION" >"${REPO_ROOT}"/collector/release/NEXT_RELEASE_VERSION