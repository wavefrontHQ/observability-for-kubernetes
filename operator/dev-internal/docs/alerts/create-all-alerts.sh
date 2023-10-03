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
  local alert_target="$4"
  local alert_files

  pushd "${TEMP_DIR}" >/dev/null
    alert_files=$(ls "${TEMP_DIR}")
    # shellcheck disable=SC2068
    for alert_file in ${alert_files[@]}; do
      check_alert_file "${alert_file}"
      post_alert_to_wavefront "${WAVEFRONT_TOKEN}" "${WF_CLUSTER}" "${K8S_CLUSTER_NAME}" "${alert_file}" "${alert_target}"
    done
  popd >/dev/null

  echo "Link to alerts just created: https://${wavefront_cluster}.wavefront.com/alerts?search=%7B%22searchTerms%22%3A%5B%7B%22type%22%3A%22tagpath%22%2C%22value%22%3A%22integration.kubernetes%22%7D%2C%7B%22type%22%3A%22freetext%22%2C%22value%22%3A%22${k8s_cluster_name}%22%7D%5D%2C%22sortOrder%22%3A%22ascending%22%2C%22sortField%22%3Anull%2C%22pageNum%22%3A1%7D&tagPathTree=%7B%22integration%22%3A%7B%22wf-value%22%3A%22integration%22%7D%7D"

}

function post_alert_to_wavefront() {
  local wavefront_token="$1"
  local wavefront_cluster="$2"
  local k8s_cluster_name="$3"
  local alert_file="$4"
  local alert_target="$5"
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
    -d @<(sed "s/K8S_CLUSTER_NAME/${k8s_cluster_name}/g" "${alert_file}" | sed "s/ALERT_TARGET/${alert_target}/g"))

  if [[ ${res_code} -ne 200 ]]; then
    print_err_and_exit "Unable to create alert: $(cat "${response}")"
  fi

  echo "Alert has been created."
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
  echo "Usage: create-all-alerts [flags] [options]"
  echo -e "\t-t wavefront api token (optional)"
  echo -e "\t-c wavefront instance name (required)"
  echo -e "\t-n kubernetes cluster name (required)"
  echo -e "\t-p end-point for csp authentication (optional)"
  echo -e "\t-a api token for csp authentication (optional)"
  echo -e "\t-i oauth app id for csp authentication (optional)"
  echo -e "\t-s oauth app secret for csp authentication (optional)"
  echo -e "\t-o oauth org id for csp authentication (optional)"
  echo -e "\t-e alert target (optional)"
  echo -e "\t-h print usage"
}

function main() {
  # Required arguments
  local WF_CLUSTER=''
  local K8S_CLUSTER_NAME=''

  # Optional arguments
  local CSP_ENDPOINT='console'
  local CSP_API_TOKEN=''
  local CSP_APP_ID=''
  local CSP_APP_SECRET=''

  # Default arguments
  local GITHUB_REPO='wavefrontHQ/observability-for-kubernetes'
  local ALERTS_DIRECTORY='docs/alerts/templates'
  local GIT_BRANCH='main'
  local ALERT_TARGET=''

  while getopts ':t:c:n:p:a:i:s:o:e:d:b:h' opt; do
    case "${opt}" in
      t) WAVEFRONT_TOKEN="${OPTARG}" ;;
      c) WF_CLUSTER="${OPTARG}" ;;
      n) K8S_CLUSTER_NAME="${OPTARG}" ;;
      p) CSP_ENDPOINT="${OPTARG}" ;;
      a) CSP_API_TOKEN="${OPTARG}" ;;
      i) CSP_APP_ID="${OPTARG}" ;;
      s) CSP_APP_SECRET="${OPTARG}" ;;
      o) CSP_ORG_ID="${OPTARG}" ;;
      e) ALERT_TARGET="${OPTARG}" ;;
      d) ALERTS_DIRECTORY="${OPTARG}" ;;
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

  # Download and create all the alerts
  TEMP_DIR=$(mktemp -d)
  download_alerts "${GITHUB_REPO}" "${ALERTS_DIRECTORY}" "${GIT_BRANCH}"
  create_alerts "${WAVEFRONT_TOKEN}" "${WF_CLUSTER}" "${K8S_CLUSTER_NAME}" "${ALERT_TARGET}"
  rm -rf "${TEMP_DIR}"
}

main "$@"
