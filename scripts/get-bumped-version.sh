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
  echo -e "\t-v version to bump (required)"
  echo -e "\t-s semver component to bump (required, ex: major, minor, patch)"
  exit 1
}

while getopts "s:v:" opt; do
  case $opt in
    v)
      VERSION="$OPTARG"
      ;;
    s)
      BUMP_COMPONENT="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
  esac
done

check_required_argument "${VERSION}" "-v <VERSION> is required"
check_required_argument "${BUMP_COMPONENT}" "-s <BUMP_COMPONENT> is required"

make -s semver-cli

semver-cli inc "$BUMP_COMPONENT" "$VERSION"