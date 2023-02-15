#!/usr/bin/env bash
set -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
OPERATOR_DIR="${REPO_ROOT}/operator"
source "${REPO_ROOT}/scripts/k8s-utils.sh"

cd "${OPERATOR_DIR}"
IMGPKG=${IMGPKG:-$OPERATOR_DIR/bin/imgpkg}

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo "expects one image ref per line on stdin"
  echo -e "\t-s source image prefix"
  echo -e "\t-d destination image prefix"
  exit 1
}

function copy-image-ref() {
    local image_ref="$1"
    local src_prefix="$2"
    local dst_prefix="$3"
    local image_name

    image_name=$(echo "$image_ref" | cut -d':' -f1)
    ${IMGPKG} copy -i "$src_prefix/$image_ref" --to-repo "$dst_prefix/$image_name"
}

function main() {
  make imgpkg
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
