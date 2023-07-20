#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

cd "$(dirname "$0")"

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-s semver component to bump for operator version (required)"
  exit 1
}

while getopts "s:" opt; do
  case $opt in
    s)
      OPERATOR_BUMP_COMPONENT="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
  esac
done

OLD_OPERATOR_VERSION=$(get_operator_version)
NEW_OPERATOR_VERSION=$(semver-cli inc "$OPERATOR_BUMP_COMPONENT" "$OLD_OPERATOR_VERSION")
echo "$NEW_OPERATOR_VERSION" >../../release/OPERATOR_VERSION