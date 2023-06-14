#!/usr/bin/env bash
set -eo pipefail

function post_alert_to_wavefront() {
  local wavefront_token=$1
  local wavefront_cluster=$2
  local alert_file=$3
  local k8s_cluster_name=$4

  response=$(mktemp)
  res_code=$(curl --silent --show-error --output "${response}" --write-out "%{http_code}" \
    -X POST "https://${wavefront_cluster}.wavefront.com/api/v2/alert?useMultiQuery=true" \
    -H "Accept: application/json" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${wavefront_token}" \
    -d @<(sed "s/K8S_CLUSTER_NAME/${k8s_cluster_name}/g" "${alert_file}"))

  if [[ ${res_code} -ne 200 ]]; then
    echo "Unable to create alert: "
    cat "${response}"
    exit 1
  fi

  if [ -x "$(command -v jq)" ]; then
    alert_name=$(jq -r '.name' "${alert_file}")
    echo "Alert name: ${alert_name}"
  fi

  alert_id=$(sed -n 's/.*id":"\([0-9]*\).*/\1/p' "${response}")

  echo "Alert has been created at: https://${wavefront_cluster}.wavefront.com/alerts/${alert_id}"
}

function check_required_argument() {
  local required_arg=$1
  local failure_msg=$2
  if [[ -z ${required_arg} ]]; then
    print_usage_and_exit "$failure_msg"
  fi
}

function print_usage_and_exit() {
  echo "Failure: $1"
  print_usage
  exit 1
}

function print_usage() {
  echo "Usage: create-alert.sh -t <WAVEFRONT_TOKEN> -c <WF_CLUSTER> -f <ALERT_FILE> -n <K8S_CLUSTER_NAME> -h"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-c wavefront instance name (required)"
  echo -e "\t-f path to alert file (required)"
  echo -e "\t-n kubernetes cluster name (required)"
  echo -e "\t-h print usage"
}

function main() {
  # Required arguments
  local WF_CLUSTER=
  local ALERT_FILE=
  local K8S_CLUSTER_NAME=
  local WF_CLUSTER=

  while getopts 'c:t:f:n:h' opt; do
    case "${opt}" in
    t) WAVEFRONT_TOKEN="${OPTARG}" ;;
    c) WF_CLUSTER="${OPTARG}" ;;
    f) ALERT_FILE="${OPTARG}" ;;
    n) K8S_CLUSTER_NAME="${OPTARG}" ;;
    h) print_usage; exit 0 ;;
    \?) print_usage_and_exit "Invalid option" ;;
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
