#!/usr/bin/env bash
set -eo pipefail

function download_alert() {
  local github_repo="$1"
  local alert_path="$2"
  local git_branch="$3"
  local alert_file="$4"
  local response res_code

  printf "Downloading alert ..."

  response=$(mktemp)
  res_code=$(curl --silent --show-error --output "${response}" --write-out "%{http_code}" \
    "https://api.github.com/repos/${github_repo}/contents/${alert_path}?ref=${git_branch}" \
    -H "Accept: application/vnd.github+json")

  if [[ ${res_code} -ne 200 ]]; then
    echo "Unable to download alert: $(cat "${response}")"
    exit 1
  fi

  local download_url
  if [ -x "$(command -v jq)" ]; then
    download_url=$(jq -r '.download_url' "${response}")
  else
    download_url=$(grep download_url "${response}" | tr '",' ' ' | awk '{print $3}')
  fi

  res_code=$(curl --silent --show-error --output "${alert_file}" --write-out "%{http_code}" -L "${download_url}")
  if [[ ${res_code} -ne 200 ]]; then
    echo "Unable to download alert: $(cat "${alert_file}")"
    exit 1
  fi

  echo " done."
}

function post_alert_to_wavefront() {
  local wavefront_token=$1
  local wavefront_cluster=$2
  local alert_file=$3
  local k8s_cluster_name=$4
  local alert_name response res_code

  if [ -x "$(command -v jq)" ]; then
    alert_name=$(jq -r '.name' "${alert_file}")
    echo "Creating alert: ${alert_name}"
  fi

  response=$(mktemp)
  res_code=$(curl --silent --show-error --output "${response}" --write-out "%{http_code}" \
    -X POST "https://${wavefront_cluster}.wavefront.com/api/v2/alert?useMultiQuery=true" \
    -H "Accept: application/json" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${wavefront_token}" \
    -d @<(sed "s/K8S_CLUSTER_NAME/${k8s_cluster_name}/g" "${alert_file}"))

  if [[ ${res_code} -ne 200 ]]; then
    echo "Unable to create alert: $(cat "${response}")"
    exit 1
  fi

  local alert_id
  alert_id=$(sed -n 's/.*id":"\([0-9]*\).*/\1/p' "${response}")

  echo "Alert has been created at: https://${wavefront_cluster}.wavefront.com/alerts/${alert_id}"
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
  echo "Usage: create-alert.sh -t <WAVEFRONT_TOKEN> -c <WF_CLUSTER> -f <ALERT_FILE_NAME> -n <K8S_CLUSTER_NAME> -h"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-c wavefront instance name (required)"
  echo -e "\t-f alert file name (required)"
  echo -e "\t-n kubernetes cluster name (required)"
  echo -e "\t-h print usage"
}

function main() {
  # Required arguments
  local WF_CLUSTER=
  local ALERT_FILE_NAME=
  local K8S_CLUSTER_NAME=

  # Default arguments
  local GITHUB_REPO='wavefrontHQ/observability-for-kubernetes'
  local ALERTS_FOLDER='docs/alerts/templates'
  local GIT_BRANCH='main'

  while getopts ':t:c:f:n:p:b:h' opt; do
    case "${opt}" in
    t) WAVEFRONT_TOKEN="${OPTARG}" ;;
    c) WF_CLUSTER="${OPTARG}" ;;
    f) ALERT_FILE_NAME="${OPTARG}" ;;
    n) K8S_CLUSTER_NAME="${OPTARG}" ;;
    p) ALERTS_FOLDER="${OPTARG}" ;;
    b) GIT_BRANCH="${OPTARG}" ;;
    h) print_usage; exit 0 ;;
    \?) print_usage_and_exit "Invalid option: -${OPTARG}" ;;
    esac
  done

  # Checking for required arguments
  check_required_argument "${WAVEFRONT_TOKEN}" "-t <WAVEFRONT_TOKEN> is required"
  check_required_argument "${WF_CLUSTER}" "-c <WF_CLUSTER> is required"
  check_required_argument "${ALERT_FILE_NAME}" "-f <ALERT_FILE_NAME> is required"
  check_required_argument "${K8S_CLUSTER_NAME}" "-n <K8S_CLUSTER_NAME> is required"

  # Download and create the alert
  TEMP_FILE=$(mktemp)
  download_alert "${GITHUB_REPO}" "${ALERTS_FOLDER}/${ALERT_FILE_NAME}" "${GIT_BRANCH}" "${TEMP_FILE}"
  check_alert_file "${TEMP_FILE}"
  post_alert_to_wavefront "${WAVEFRONT_TOKEN}" "${WF_CLUSTER}" "${TEMP_FILE}" "${K8S_CLUSTER_NAME}"
  rm "${TEMP_FILE}"
}

main "$@"
