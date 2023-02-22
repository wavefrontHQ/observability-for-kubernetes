#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"



while getopts "o:c:" opt; do
  case $opt in
    o)
      OPERATOR_BUMP_COMPONENT="$OPTARG"
      ;;
    c)
      COLLECTOR_BUMP_COMPONENT="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
  esac
done

OLD_OPERATOR_VERSION=$(get_operator_version)
OPERATOR_VERSION="$("${REPO_ROOT}"/scripts/get-bumped-version.sh -v "${OLD_OPERATOR_VERSION}" -s "${OPERATOR_BUMP_COMPONENT}")"
echo "$OPERATOR_VERSION" >"${REPO_ROOT}"/operator/release/OPERATOR_VERSION

OLD_COLLECTOR_VERSION=$(cat "${REPO_ROOT}"/collector/release/VERSION)
COLLECTOR_VERSION="$("${REPO_ROOT}"/scripts/get-bumped-version.sh -v "${OLD_COLLECTOR_VERSION}" -s "${COLLECTOR_BUMP_COMPONENT}")"
echo "$COLLECTOR_VERSION" >"${REPO_ROOT}"/collector/release/VERSION
#echo "$COLLECTOR_VERSION" >"${REPO_ROOT}"/collector/release/VERSION

git show origin/rc:operator/wavefront-operator-main.yaml > ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml
OPERATOR_ALPHA_TAG=$(cat "${REPO_ROOT}"/operator/dev-internal/deploy/wavefront-operator.yaml | yq 'select(.metadata.name == "wavefront-controller-manager" and .kind == "Deployment" ) | .spec.template.spec.containers[0].image' | cut -d ':' -f2)
COLLECTOR_ALPHA_TAG=$(cat "${REPO_ROOT}"/operator/dev-internal/deploy/wavefront-operator.yaml | yq 'select(.metadata.name == "wavefront-component-versions" ) | .data.collector')
crane copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-operator:${OPERATOR_ALPHA_TAG}" "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-operator-snapshot:${OPERATOR_VERSION}"
crane copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-collector:${COLLECTOR_ALPHA_TAG}" "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-collector-snapshot:${COLLECTOR_VERSION}"
