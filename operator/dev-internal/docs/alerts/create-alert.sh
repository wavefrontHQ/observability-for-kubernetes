#!/usr/bin/env bash
set -eo pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"

source "${REPO_ROOT}/scripts/k8s-utils.sh"

function post_alert_to_wavefront() {
  local wavefront_token=$1
  local wavefront_cluster=$2
  local alert_file=$3
  local k8s_cluster_name=$4

  response=$(mktemp)
  res_code=$(curl -X POST --silent --output "${response}" --write-out "%{http_code}" \
    "https://${wavefront_cluster}.wavefront.com/api/v2/alert?useMultiQuery=true" \
    -H "Accept: application/json" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${wavefront_token}" \
    -d @<(sed "s/K8S_CLUSTER_NAME/${k8s_cluster_name}/g" "${alert_file}"))

  if [[ ${res_code} -ne 200 ]]; then
    red "Unable to create alert: "
    cat "${response}"
    exit 1
  fi

  alert_id=$(sed -n 's/.*id":"\([0-9]*\).*/\1/p' "${response}")

  green "Alert has been created at: https://${wavefront_cluster}.wavefront.com/alerts/${alert_id}"
}
function check_required_argument() {
  local required_arg=$1
  local failure_msg=$2
  if [[ -z ${required_arg} ]]; then
    print_usage_and_exit "$failure_msg"
  fi
}

function print_usage_and_exit() {
  red "Failure: $1"
  echo "Usage: $0 -t <WAVEFRONT_TOKEN> -c <WF_CLUSTER> -f <ALERT_FILE> -n <K8S_CLUSTER_NAME>"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-c wavefront instance name (required)"
  echo -e "\t-f alert file (required)"
  echo -e "\t-n kubernetes cluster name (required)"
  exit 1
}

function main() {
  # Required arguments
  local WF_CLUSTER=
  local ALERT_FILE=
  local K8S_CLUSTER_NAME=
  local WF_CLUSTER=

  while getopts 'c:t:f:n:' opt; do
    case "${opt}" in
    t) WAVEFRONT_TOKEN="${OPTARG}" ;;
    c) WF_CLUSTER="${OPTARG}" ;;
    f) ALERT_FILE="${OPTARG}" ;;
    n) K8S_CLUSTER_NAME="${OPTARG}" ;;
    \?) print_usage_and_exit "Invalid option: -${OPTARG}" ;;
    esac
  done

  # Checking for required arguments
  check_required_argument "${WAVEFRONT_TOKEN}" "-t <WAVEFRONT_TOKEN> is required"
  check_required_argument "${WF_CLUSTER}" "-c <WF_CLUSTER> is required"
  check_required_argument "${ALERT_FILE}" "-f <ALERT_FILE> is required"
  check_required_argument "${K8S_CLUSTER_NAME}" "-n <K8S_CLUSTER_NAME> is required"

  post_alert_to_wavefront "${WAVEFRONT_TOKEN}" "${WF_CLUSTER}" "${ALERT_FILE}" "${K8S_CLUSTER_NAME}"
}

main "$@"
