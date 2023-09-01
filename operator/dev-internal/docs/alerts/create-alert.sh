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

function check_alert_path() {
  local alert_path=$1

  if ! [ -f "${alert_path}" ] && ! [ -d "${alert_path}" ]; then
    echo "Invalid alert path: ${alert_path}"
    exit 1
  fi

  # Alert Path is a directory
  if [ -d "${alert_path}" ]; then
    for alert_file in "${alert_path}"/*; do
      [ -f "${alert_file}" ] && check_alert_file "${alert_file}"
    done
  fi

  # Alert Path is a file
  if [ -f "${alert_path}" ]; then
    check_alert_file "${alert_path}"
  fi
}

function check_alert_file() {
  local alert_file=$1

  if ! [ -f "${alert_file}" ]; then
    echo "Invalid alert file: ${alert_file}"
    exit 1
  fi

  if [ -x "$(command -v jq)" ] && ! jq -e . "${alert_file}" &>/dev/null; then
    echo "Invalid json format for alert file: ${alert_file}"
    exit 1
  elif [ -x "$(command -v python)" ] \
    && ! python -c "import sys,json;json.loads(sys.stdin.read())" < "${alert_file}" &>/dev/null; then
    echo "Invalid json format for alert file: ${alert_file}"
    exit 1
  elif [ -x "$(command -v python3)" ] \
    && ! python3 -c "import sys,json;json.loads(sys.stdin.read())" < "${alert_file}" &>/dev/null; then
    echo "Invalid json format for alert file: ${alert_file}"
    exit 1
  fi
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
  echo "Usage: create-alert.sh -t <WAVEFRONT_TOKEN> -c <WF_CLUSTER> -f <ALERT_PATH> -n <K8S_CLUSTER_NAME> -h"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-c wavefront instance name (required)"
  echo -e "\t-f path to alert file or alert folder(required)"
  echo -e "\t-n kubernetes cluster name (required)"
  echo -e "\t-h print usage"
}

function main() {
  # Required arguments
  local WF_CLUSTER=
  local ALERT_PATH=
  local K8S_CLUSTER_NAME=

  while getopts 'c:t:f:n:h' opt; do
    case "${opt}" in
    t) WAVEFRONT_TOKEN="${OPTARG}" ;;
    c) WF_CLUSTER="${OPTARG}" ;;
    f) ALERT_PATH="${OPTARG}" ;;
    n) K8S_CLUSTER_NAME="${OPTARG}" ;;
    h) print_usage; exit 0 ;;
    \?) print_usage_and_exit "Invalid option" ;;
    esac
  done

  # Checking for required arguments
  check_required_argument "${WAVEFRONT_TOKEN}" "-t <WAVEFRONT_TOKEN> is required"
  check_required_argument "${WF_CLUSTER}" "-c <WF_CLUSTER> is required"
  check_required_argument "${ALERT_PATH}" "-f <ALERT_PATH> is required"
  check_required_argument "${K8S_CLUSTER_NAME}" "-n <K8S_CLUSTER_NAME> is required"

  check_alert_path "${ALERT_PATH}"

  if [ -f "${ALERT_PATH}" ]; then
      post_alert_to_wavefront "${WAVEFRONT_TOKEN}" "${WF_CLUSTER}" "${ALERT_PATH}" "${K8S_CLUSTER_NAME}"
  fi

  if [ -d "${ALERT_PATH}" ]; then
    for alert_file in "${ALERT_PATH}"/*; do
      [ -f "${alert_file}" ] && post_alert_to_wavefront "${WAVEFRONT_TOKEN}" "${WF_CLUSTER}" "${alert_file}" "${K8S_CLUSTER_NAME}"
    done

  fi

}

main "$@"
