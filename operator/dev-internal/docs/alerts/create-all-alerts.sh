#!/usr/bin/env bash
set -eo pipefail

function download_alerts() {
  local github_repo="$1"
  local alerts_path="$2"
  local git_branch="$3"
  local response res_code

  printf "Downloading alerts ..."

  response=$(mktemp)
  res_code=$(curl --silent --show-error --output "${response}" --write-out "%{http_code}" \
    "https://api.github.com/repos/${github_repo}/contents/${alerts_path}?ref=${git_branch}" \
    -H "Accept: application/vnd.github+json")

  if [[ ${res_code} -ne 200 ]]; then
    print_err_and_exit "Unable to download alerts: $(cat "${response}")"
  fi

  pushd "${TEMP_DIR}" >/dev/null
    local download_urls
    if [ -x "$(command -v jq)" ]; then
      download_urls=$(jq -r '.[].download_url' "${response}")
    else
      download_urls=$(grep download_url "${response}" | awk '{print $2}' | tr '",' ' ')
    fi
    # shellcheck disable=SC2068
    for download_url in ${download_urls[@]}; do
      res_code=$(curl --silent --show-error --write-out "%{http_code}" -LO "${download_url}")
      if [[ ${res_code} -ne 200 ]]; then
        print_err_and_exit "Unable to download alert at: ${download_url}"
      fi
    done
  popd >/dev/null

  echo " done."
}

function create_alerts() {
  local wavefront_token="$1"
  local wavefront_cluster="$2"
  local k8s_cluster_name="$3"
  local alert_files

  pushd "${TEMP_DIR}" >/dev/null
    alert_files=$(ls "${TEMP_DIR}")
    # shellcheck disable=SC2068
    for alert_file in ${alert_files[@]}; do
      check_alert_file "${alert_file}"
      post_alert_to_wavefront "${WAVEFRONT_TOKEN}" "${WF_CLUSTER}" "${K8S_CLUSTER_NAME}" "${alert_file}"
    done
  popd >/dev/null

  echo "Finished creating alerts."
}

function post_alert_to_wavefront() {
  local wavefront_token="$1"
  local wavefront_cluster="$2"
  local k8s_cluster_name="$3"
  local alert_file="$4"
  local alert_name response res_code

  if [ -x "$(command -v jq)" ]; then
    alert_name=$(jq -r '.name' "${alert_file}")
    echo "Creating alert: ${alert_name}"
  else
    echo "Creating alert: ${alert_file}"
  fi

  response=$(mktemp)
  res_code=$(curl --silent --show-error --output "${response}" --write-out "%{http_code}" \
    -X POST "https://${wavefront_cluster}.wavefront.com/api/v2/alert?useMultiQuery=true" \
    -H "Accept: application/json" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${wavefront_token}" \
    -d @<(sed "s/K8S_CLUSTER_NAME/${k8s_cluster_name}/g" "${alert_file}"))

  if [[ ${res_code} -ne 200 ]]; then
    print_err_and_exit "Unable to create alert: $(cat "${response}")"
  fi

  local alert_id
  alert_id=$(sed -n 's/.*id":"\([0-9]*\).*/\1/p' "${response}")

  echo "Alert has been created at: https://${wavefront_cluster}.wavefront.com/alerts/${alert_id}"
}

function check_alert_file() {
  local alert_file="$1"

  if ! [ -f "${alert_file}" ]; then
    print_err_and_exit "Invalid alert file: ${alert_file}"
  fi

  if [ -x "$(command -v jq)" ] && ! jq -e . "${alert_file}" &>/dev/null; then
    print_err_and_exit "Invalid json format for alert file: ${alert_file}"
  elif [ -x "$(command -v python)" ] \
    && ! python -c "import sys,json;json.loads(sys.stdin.read())" < "${alert_file}" &>/dev/null; then
    print_err_and_exit "Invalid json format for alert file: ${alert_file}"
  elif [ -x "$(command -v python3)" ] \
    && ! python3 -c "import sys,json;json.loads(sys.stdin.read())" < "${alert_file}" &>/dev/null; then
    print_err_and_exit "Invalid json format for alert file: ${alert_file}"
  fi
}

function check_required_argument() {
  local required_arg="$1"
  local failure_msg="$2"
  if [[ -z "${required_arg}" ]]; then
    print_usage_and_exit "${failure_msg}"
  fi
}

function print_err_and_exit() {
  echo "Error: $1"
  exit 1
}

function print_usage_and_exit() {
  echo "Failure: $1"
  print_usage
  exit 1
}

function print_usage() {
  echo "Usage: create-all-alerts.sh -t <WAVEFRONT_TOKEN> -c <WF_CLUSTER> -n <K8S_CLUSTER_NAME> -h"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-c wavefront instance name (required)"
  echo -e "\t-n kubernetes cluster name (required)"
  echo -e "\t-h print usage"
}

function main() {
  # Required arguments
  local WF_CLUSTER=''
  local K8S_CLUSTER_NAME=''

  # Default arguments
  local GITHUB_REPO='wavefrontHQ/observability-for-kubernetes'
  local ALERTS_FOLDER='docs/alerts/templates'
  local GIT_BRANCH='main'

  while getopts 'c:t:n:f:b:h' opt; do
    case "${opt}" in
      t) WAVEFRONT_TOKEN="${OPTARG}" ;;
      c) WF_CLUSTER="${OPTARG}" ;;
      n) K8S_CLUSTER_NAME="${OPTARG}" ;;
      f) ALERTS_FOLDER="${OPTARG}" ;;
      b) GIT_BRANCH="${OPTARG}" ;;
      h) print_usage; exit 0 ;;
      \?) print_usage_and_exit "Invalid option" ;;
    esac
  done

  # Checking for required arguments
  check_required_argument "${WAVEFRONT_TOKEN}" "-t <WAVEFRONT_TOKEN> is required"
  check_required_argument "${WF_CLUSTER}" "-c <WF_CLUSTER> is required"
  check_required_argument "${K8S_CLUSTER_NAME}" "-n <K8S_CLUSTER_NAME> is required"

  # Download and create all the alerts
  TEMP_DIR=$(mktemp -d)
  download_alerts "${GITHUB_REPO}" "${ALERTS_FOLDER}" "${GIT_BRANCH}"
  create_alerts "${WAVEFRONT_TOKEN}" "${WF_CLUSTER}" "${K8S_CLUSTER_NAME}"
  rm -rf "${TEMP_DIR}"
}

main "$@"
