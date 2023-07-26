#!/usr/bin/env bash
set -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
OPERATOR_DIR="${REPO_ROOT}/operator"
source "${REPO_ROOT}/scripts/k8s-utils.sh"

cd "${OPERATOR_DIR}"

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo "Copies a mutli-platform image bundle from one registry to another."
  echo "Expects one image ref per line on stdin. Each line should look like 'kubernetes-operator:2.2.1-alpha-57321970'."
  echo -e "\t-s source image prefix (like projects.registry.vmware.com/tanzu_observability_keights_saas)"
  echo -e "\t-d destination image prefix (like projects.registry.vmware.com/tanzu_observability)"
  exit 1
}

function copy-image-ref() {
    local image_ref="$1"
    local src_prefix="$2"
    local dst_prefix="$3"
    local image_name

    image_name=$(echo "$image_ref" | cut -d':' -f1)
    imgpkg copy -i "$src_prefix/$image_ref" --to-repo "$dst_prefix/$image_name"
}

function main() {
  local src_prefix=
  local dst_prefix=

  while getopts ":s:d:" opt; do
    case $opt in
    s)
      src_prefix="$OPTARG"
      ;;
    d)
      dst_prefix="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ -z "$src_prefix" ]]; then
    print_usage_and_exit "-s required"
  fi

  if [[ -z "$dst_prefix" ]]; then
    print_usage_and_exit "-d required"
  fi

  while IFS='$\n' read -r image_ref; do
      copy-image-ref "$image_ref" "$src_prefix" "$dst_prefix"
  done
}

main "$@"
