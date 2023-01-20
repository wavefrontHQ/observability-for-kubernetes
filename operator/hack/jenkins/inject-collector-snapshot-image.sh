#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

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

  check_required_argument "$REGISTRY_NAME" "-r <REGISTRY_NAME> is required"
  check_required_argument "$IMAGE_NAME" "-n <IMAGE_NAME> is required"
  check_required_argument "$VERSION_POSTFIX" "-v <VERSION_POSTFIX> is required"

  local current_version="$(cat collector/release/VERSION)"
  local bumped_version=$(./scripts/get-bumped-version.sh -v "${current_version}" -s patch)
  local image_version="${bumped_version}${VERSION_POSTFIX}"
  local image="${REGISTRY_NAME}/${IMAGE_NAME}:${image_version}"

  cp operator/deploy/internal/collector/3-wavefront-collector-deployment.yaml operator/deploy/internal/collector/3-wavefront-collector-deployment.yaml.bak
  cp operator/deploy/internal/collector/2-wavefront-collector-daemonset.yaml operator/deploy/internal/collector/2-wavefront-collector-daemonset.yaml.bak
	sed -i "s%image:.*kubernetes-collector:.*$%image: ${image}%" operator/deploy/internal/collector/3-wavefront-collector-deployment.yaml
	sed -i "s%image:.*kubernetes-collector:.*$%image: ${image}%" operator/deploy/internal/collector/2-wavefront-collector-daemonset.yaml
}

main "$@"