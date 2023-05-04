#!/usr/bin/env bash
set -ex

REPO_ROOT=$(git rev-parse --show-toplevel)
OPERATOR_BUILD_DIR="${REPO_ROOT}/operator/build/operator"
NS="observability-system"
PREFIX="projects.registry.vmware.com/tanzu_observability_keights_saas"

source "${REPO_ROOT}/scripts/k8s-utils.sh"

function main() {
  local BUILD_COLLECTOR=true
  local BUILD_OPERATOR=true
  local COLLECTOR_VERSION="$(get_component_version collector)"

  while getopts ":c:o" opt; do
    case $opt in
    c)
      BUILD_COLLECTOR=false
      ;;
    o)
      BUILD_OPERATOR=false
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ "${BUILD_COLLECTOR}" == "true" ]]; then
    pushd ${REPO_ROOT}/collector
      local output="$(PREFIX=${PREFIX} make docker-xplatform-build)"
      COLLECTOR_VERSION="${output#*Built collector version: }"
    popd
  fi

  if [[ "${BUILD_OPERATOR}" == "true" ]]; then
    pushd ${REPO_ROOT}/operator
      PREFIX=${PREFIX} make operator-yaml
    popd
  fi

  sed -i.bak "s%collector:.*$%collector: ${COLLECTOR_VERSION}%" "${OPERATOR_BUILD_DIR}/wavefront-operator.yaml"

  kubectl apply -k ${OPERATOR_BUILD_DIR}
  kubectl create -n ${NS} secret generic wavefront-secret --from-literal token=${WAVEFRONT_TOKEN} || true
}

main "$@"
