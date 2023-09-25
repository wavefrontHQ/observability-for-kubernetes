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
    print_err_and_exit "Unable to download alert: $(cat "${response}")"
  fi

  local download_url
  if [ -x "$(command -v jq)" ]; then
    download_url=$(jq -r '.download_url' "${response}")
  else
    download_url=$(grep download_url "${response}" | tr '",' ' ' | awk '{print $3}')
  fi

  res_code=$(curl --silent --show-error --output "${alert_file}" --write-out "%{http_code}" -L "${download_url}")
  if [[ ${res_code} -ne 200 ]]; then
    print_err_and_exit "Unable to download alert: $(cat "${alert_file}")"
  fi

  echo " done."
}

function post_alert_to_wavefront() {
  local wavefront_token="$1"
  local wavefront_cluster="$2"
  local alert_file="$3"
  local k8s_cluster_name="$4"
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

function get_csp_access_token() {
  local csp_endpoint="$1"
  local csp_token_or_secret="$2"
  local csp_app_id="$3"
  local csp_org_id="$4"
  local csp_access_token response res_code

  printf "Retrieving the CSP access token ..."

  response=$(mktemp)

  if [[ -z "${csp_app_id}" ]]; then
    local csp_api_token="${csp_token_or_secret}"
    res_code=$(curl --silent --show-error --output "${response}" --write-out "%{http_code}" \
      -X POST "https://${csp_endpoint}.cloud.vmware.com/csp/gateway/am/api/auth/api-tokens/authorize" \
      -H "Accept: application/json" \
      -H "Content-Type: application/x-www-form-urlencoded" \
      -d "api_token=${csp_api_token}")
  else
    local csp_credentials
    csp_credentials=$(printf '%s:%s' "${csp_app_id}" "${csp_token_or_secret}" | base64)
    res_code=$(curl --silent --show-error --output "${response}" --write-out "%{http_code}" \
      -X POST "https://${csp_endpoint}.cloud.vmware.com/csp/gateway/am/api/auth/authorize" \
      -H "Accept: application/json" \
      -H "Content-Type: application/x-www-form-urlencoded" \
      -H "Authorization: Basic ${csp_credentials}" \
      -d "grant_type=client_credentials&orgId=${csp_org_id}")
  fi

  if [ -x "$(command -v jq)" ]; then
    if [[ ${res_code} -ne 200 ]]; then
      print_err_and_exit "Unable to retrieve the CSP access token: $(jq -r '.message' "${response}")"
    fi
    csp_access_token=$(jq -r '.access_token' "${response}")
  else
    if [[ ${res_code} -ne 200 ]]; then
      print_err_and_exit "Unable to retrieve the CSP access token: $(cat "${response}")"
    fi
    for item in $(tr '{,}' ' ' < "${response}"); do
      if echo "${item}" | grep access_token >/dev/null; then
        csp_access_token=$(echo "${item}" | tr '"' ' ' | awk '{print $3}')
        break
      fi
    done
  fi

  WAVEFRONT_TOKEN="${csp_access_token}"

  echo " done."
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
  echo "Usage: create-alert [flags] [options]"
  echo -e "\t-t wavefront api token (optional)"
  echo -e "\t-c wavefront instance name (required)"
  echo -e "\t-n kubernetes cluster name (required)"
  echo -e "\t-f alert template file name (required)"
  echo -e "\t-e end-point for csp authentication (optional)"
  echo -e "\t-a api token for csp authentication (optional)"
  echo -e "\t-i oauth app id for csp authentication (optional)"
  echo -e "\t-s oauth app secret for csp authentication (optional)"
  echo -e "\t-o oauth org id for csp authentication (optional)"
  echo -e "\t-h print usage"
}

function main() {
  # Required arguments
  local WF_CLUSTER=''
  local ALERT_FILE_NAME=''
  local K8S_CLUSTER_NAME=''

  # Optional arguments
  local CSP_ENDPOINT='console'
  local CSP_API_TOKEN=''
  local CSP_APP_ID=''
  local CSP_APP_SECRET=''

  # Default arguments
  local GITHUB_REPO='wavefrontHQ/observability-for-kubernetes'
  local ALERTS_FOLDER='docs/alerts/templates'
  local GIT_BRANCH='main'

  while getopts ':t:c:n:f:e:a:i:s:o:p:b:h' opt; do
    case "${opt}" in
      t) WAVEFRONT_TOKEN="${OPTARG}" ;;
      c) WF_CLUSTER="${OPTARG}" ;;
      n) K8S_CLUSTER_NAME="${OPTARG}" ;;
      f) ALERT_FILE_NAME="${OPTARG}" ;;
      e) CSP_ENDPOINT="${OPTARG}" ;;
      a) CSP_API_TOKEN="${OPTARG}" ;;
      i) CSP_APP_ID="${OPTARG}" ;;
      s) CSP_APP_SECRET="${OPTARG}" ;;
      o) CSP_ORG_ID="${OPTARG}" ;;
      p) ALERTS_FOLDER="${OPTARG}" ;;
      b) GIT_BRANCH="${OPTARG}" ;;
      h) print_usage; exit 0 ;;
      \?) print_usage_and_exit "Invalid option: -${OPTARG}" ;;
    esac
  done

  # Get the CSP access token if necessary
  if [[ -n "${CSP_API_TOKEN}" ]]; then
    get_csp_access_token "${CSP_ENDPOINT}" "${CSP_API_TOKEN}"
  elif [[ -n "${CSP_APP_ID}" ]]; then
    get_csp_access_token "${CSP_ENDPOINT}" "${CSP_APP_SECRET}" "${CSP_APP_ID}" "${CSP_ORG_ID}"
  fi

  # Checking for required arguments
  check_required_argument "${WAVEFRONT_TOKEN}" "-t <WAVEFRONT_TOKEN> is required"
  check_required_argument "${WF_CLUSTER}" "-c <WF_CLUSTER> is required"
  check_required_argument "${K8S_CLUSTER_NAME}" "-n <K8S_CLUSTER_NAME> is required"
  check_required_argument "${ALERT_FILE_NAME}" "-f <ALERT_FILE_NAME> is required"

  # Download and create the alert
  TEMP_FILE=$(mktemp)
  download_alert "${GITHUB_REPO}" "${ALERTS_FOLDER}/${ALERT_FILE_NAME}" "${GIT_BRANCH}" "${TEMP_FILE}"
  check_alert_file "${TEMP_FILE}"
  post_alert_to_wavefront "${WAVEFRONT_TOKEN}" "${WF_CLUSTER}" "${TEMP_FILE}" "${K8S_CLUSTER_NAME}"
  rm "${TEMP_FILE}"
}

main "$@"
