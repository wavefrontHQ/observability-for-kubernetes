#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

excludedKeys=$(jq -r '.exclude | join(", ")' ${REPO_ROOT}/scripts/dashboard-development/key-filter.json)

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-c wavefront instance name (default: 'qa4')"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-d dashboard url (required)"
  exit 1
}

function main() {
  cd "${REPO_ROOT}/scripts/dashboard-development/working"

  # REQUIRED
  local WAVEFRONT_TOKEN=
  local DASHBOARD_URL=

  local WF_CLUSTER=qa4
  local DASHBOARD_OUTPUT_FILE=

  while getopts ":c:t:d:o:" opt; do
    case $opt in
    c)
      WF_CLUSTER="$OPTARG"
      ;;
    t)
      WAVEFRONT_TOKEN="$OPTARG"
      ;;
    d)
      DASHBOARD_URL="$OPTARG"
      ;;
    o)
      DASHBOARD_OUTPUT_FILE="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ -z ${WAVEFRONT_TOKEN} ]]; then
    print_msg_and_exit "wavefront token required"
  fi

  if [[ -z ${DASHBOARD_URL} ]]; then
    print_msg_and_exit "dashboard url required"
  fi

  if [[ -z ${DASHBOARD_OUTPUT_FILE} ]]; then
    print_msg_and_exit "dashboard output file required"
  fi

  curl -sX GET --header "Accept: application/json" \
    --header "Authorization: Bearer ${WAVEFRONT_TOKEN}" \
    "https://${WF_CLUSTER}.wavefront.com/api/v2/dashboard/${DASHBOARD_URL}" \
    | jq "del(.response | ${excludedKeys})"  | jq .response \
    > "${DASHBOARD_OUTPUT_FILE}"
}

main $@
