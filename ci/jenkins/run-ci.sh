#!/usr/bin/env bash
set -e

function check_for_ci_changes() {
  local base="$1"
  local target="$2"
  local files="$3"

  if git diff --diff-filter=ADMR --name-only ${base}..${target} -- ${files} \
    | grep -v operator/dev-internal &>/dev/null; then
    echo true
  else
    echo false
  fi
}

function check_required_argument() {
  local required_arg="$1"
  local failure_msg="$2"

  if [[ -z "${required_arg}" ]]; then
    print_usage_and_exit "${failure_msg}"
  fi
}

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 -b <BASE_COMMIT> -d <TARGET_COMMIT> -f <FILE_PATHS>"
  echo -e "\t-b base commit or branch (required, ex: origin/main, HEAD~)"
  echo -e "\t-d target commit or branch (required, ex: b50ca31b, HEAD)"
  echo -e "\t-f file paths to check (required, ex: 'collector operator ci.Jenkinsfile')"
  exit 1
}

function main() {
  # Required args
  local base_commit=''
  local target_commit=''
  local file_paths=''

  while getopts 'b:d:f:' opt; do
    case "${opt}" in
      b) base_commit="${OPTARG}" ;;
      d) target_commit="${OPTARG}" ;;
      f) file_paths="${OPTARG}" ;;
      \?) print_usage_and_exit "Invalid option" ;;
    esac
  done

  check_required_argument "${base_commit}" "-b <BASE_COMMIT> is required"
  check_required_argument "${target_commit}" "-d <TARGET_COMMIT> is required"
  check_required_argument "${file_paths}" "-f <FILE_PATHS> is required"

  # returns true, if there are 'added', 'deleted', 'modified', or 'removed' changes
  # returns false, if there are no changes or changes are only in operator/dev-internal
  check_for_ci_changes "${base_commit}" "${target_commit}" "${file_paths}"
}

main "$@"
