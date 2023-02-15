#!/usr/bin/env bash
set -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
OPERATOR_DIR="${REPO_ROOT}/operator"

function check_required_argument() {
  local required_arg=$1
  local failure_msg=$2
  if [[ -z ${required_arg} ]]; then
    print_usage_and_exit "$failure_msg"
  fi
}

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-r image registry name (required, ex: dockerhub.com)"
  echo -e "\t-n image name (required, ex: kubernetes-collector)"
  echo -e "\t-v version postfix (required, -alpha-<some-sha>)"
  exit 1
}

function main() {
  cd "${REPO_ROOT}"

  REGISTRY_NAME=''
  IMAGE_NAME=''
  VERSION_POSTFIX=''

  local current_version
  local bumped_version

  while getopts ":r:n:v:" opt; do
    case $opt in
    r)
      REGISTRY_NAME="$OPTARG"
      ;;
    n)
      IMAGE_NAME="$OPTARG"
      ;;
    v)
      VERSION_POSTFIX="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  check_required_argument "${REGISTRY_NAME}" "-r <REGISTRY_NAME> is required"
  check_required_argument "${IMAGE_NAME}" "-n <IMAGE_NAME> is required"
  check_required_argument "${VERSION_POSTFIX}" "-v <VERSION_POSTFIX> is required"

  current_version="$(yq .data.collector "${OPERATOR_DIR}/config/manager/component_versions.yaml")"
  bumped_version="$("${REPO_ROOT}"/scripts/get-bumped-version.sh -v "${current_version}" -s patch)"
  local image_version="${bumped_version}${VERSION_POSTFIX}"
  local image="${REGISTRY_NAME}/${IMAGE_NAME}:${image_version}"

	sed -i.bak "s%image:.*kubernetes-collector.*$%image: ${image}%" "${OPERATOR_DIR}"/deploy/internal/collector/3-wavefront-collector-deployment.yaml
	sed -i.bak "s%image:.*kubernetes-collector.*$%image: ${image}%" "${OPERATOR_DIR}"/deploy/internal/collector/2-wavefront-collector-daemonset.yaml
}

main "$@"
