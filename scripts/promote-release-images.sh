#!/bin/bash
set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

function check_required_argument() {
  local required_arg=$1
  local failure_msg=$2
  if [[ -z ${required_arg} ]]; then
    print_usage_and_exit "$failure_msg"
  fi
}

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Promotes Operator and Collector images to releasable version."
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-o semver component to bump for operator version (required)"
  echo -e "\t-c semver component to bump for collector version (required)"
  exit 1
}

while getopts ":o:c:" opt; do
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

check_required_argument "${OPERATOR_BUMP_COMPONENT}" "-o <OPERATOR_BUMP_COMPONENT> is required"
check_required_argument "${COLLECTOR_BUMP_COMPONENT}" "-c <COLLECTOR_BUMP_COMPONENT> is required"

# Bump versions
OLD_OPERATOR_VERSION=$(get_operator_version)
OPERATOR_VERSION="$("${REPO_ROOT}"/scripts/get-bumped-version.sh -v "${OLD_OPERATOR_VERSION}" -s "${OPERATOR_BUMP_COMPONENT}")-test"
echo "$OPERATOR_VERSION" >"${REPO_ROOT}"/operator/release/OPERATOR_VERSION

OLD_COLLECTOR_VERSION=$(cat "${REPO_ROOT}"/collector/release/VERSION)
COLLECTOR_VERSION="$("${REPO_ROOT}"/scripts/get-bumped-version.sh -v "${OLD_COLLECTOR_VERSION}" -s "${COLLECTOR_BUMP_COMPONENT}")-test"
echo "$COLLECTOR_VERSION" >"${REPO_ROOT}"/collector/release/VERSION
yq -i e ".data.collector |= \"$COLLECTOR_VERSION\"" "${REPO_ROOT}"/operator/config/manager/component_versions.yaml

# Copy last tested wavefront-operator yaml from rc branch to dev-internal
git show origin/rc:operator/wavefront-operator-main.yaml > ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml

# Promote alpha images to release versions
OPERATOR_ALPHA_IMAGE=$(cat "${REPO_ROOT}"/operator/dev-internal/deploy/wavefront-operator.yaml | yq 'select(.metadata.name == "wavefront-controller-manager" and .kind == "Deployment" ) | .spec.template.spec.containers[0].image')
OPERATOR_ALPHA_TAG=$(echo ${OPERATOR_ALPHA_IMAGE} | cut -d ':' -f2)
COLLECTOR_ALPHA_TAG=$(cat "${REPO_ROOT}"/operator/dev-internal/deploy/wavefront-operator.yaml | yq 'select(.metadata.name == "wavefront-component-versions" ) | .data.collector')
crane -v copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-operator:${OPERATOR_ALPHA_TAG}" "projects.registry.vmware.com/tanzu_observability/kubernetes-operator:${OPERATOR_VERSION}"
crane -v copy "projects.registry.vmware.com/tanzu_observability_keights_saas/kubernetes-collector:${COLLECTOR_ALPHA_TAG}" "projects.registry.vmware.com/tanzu_observability/kubernetes-collector:${COLLECTOR_VERSION}"

# Update wavefront-operator yaml in dev-internal with release versions
sed -i.bak "s/collector: ${COLLECTOR_ALPHA_TAG}/collector: ${COLLECTOR_VERSION}/g" ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml
sed -i.bak "s#image: ${OPERATOR_ALPHA_IMAGE}#image: projects.registry.vmware.com/tanzu_observability/kubernetes-operator:${OPERATOR_VERSION}#g" ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml
rm -rf ${REPO_ROOT}/operator/dev-internal/deploy/wavefront-operator.yaml.bak

# Promote dev-internal to top level
pushd ${REPO_ROOT}
  make promote-internal
popd
