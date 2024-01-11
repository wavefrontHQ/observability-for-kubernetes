#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

COLLECTOR_REPO_ROOT=$(git rev-parse --show-toplevel)/collector
SCRIPT_DIR=$(cd "$(dirname "$0")" && pwd)
NS=wavefront-collector

function curl_query_to_wf_dashboard() {
  local query=$1
  local AFTER_UNIX_TS
  AFTER_UNIX_TS="$(date '+%s')000"

  # NOTE: any output inside this function is concatenated and used as the return value;
  # otherwise we would love to put a log such as this in here to give us more information:
  # echo "=============== Querying '$WF_CLUSTER' for query '${query}'"
  curl --silent --show-error --get \
    --header "Accept: application/json" \
    --header "Authorization: Bearer $WAVEFRONT_TOKEN" \
    --data-urlencode "q=$query" \
    --data-urlencode "queryType=WQL" \
    --data-urlencode "s=$AFTER_UNIX_TS" \
    --data-urlencode "g=s" \
    --data-urlencode "view=METRIC" \
    --data-urlencode "sorted=false" \
    --data-urlencode "cached=true" \
    --data-urlencode "useRawQK=false" \
    "https://$WF_CLUSTER.wavefront.com/api/v2/chart/api" |
    jq '.timeseries[0].data[0][1]'
}

function wait_for_query_match_exact() {
  local expected=$1
  shift
  local query_match_exact="$*"
  local actual
  local loop_count=0

  printf "checking wavefront query '%s' matches %s (max-queries=%s):" "$query_match_exact" "$expected" "$MAX_QUERY_TIMES"

  while [[ $loop_count -lt $MAX_QUERY_TIMES ]]; do
    loop_count=$((loop_count + 1))

    actual=$(curl_query_to_wf_dashboard "$query_match_exact" | awk '{printf "%.3f", $1}')

    if echo "$actual $expected" | awk '{exit ($1 > $2 || $1 < $2)}'; then
      green " pass."
      return 0
    fi

    printf " got %s" "$actual"
    sleep $CURL_WAIT
  done

  red " fail."
  return 1
}

function wait_for_query_non_zero() {
  local query_non_zero="$*"
  local actual=0
  local loop_count=0

  printf "checking wavefront query '%s' returns non-zero result (max-queries=%s):" "$query_non_zero" "$MAX_QUERY_TIMES"
  while [[ $loop_count -lt $MAX_QUERY_TIMES ]]; do
    loop_count=$((loop_count + 1))


    actual=$(curl_query_to_wf_dashboard "$query_non_zero")

    if ! [[ $actual == null || $actual == 0 ]]; then
      green " pass."
      return 0
    fi

    printf " got %s" "$actual"
    sleep $CURL_WAIT
  done

  red " fail."
  return 1
}

function print_usage_and_exit() {
  red "Failure: $1"
  echo "Usage: $0 -c <WAVEFRONT_CLUSTER> -t <WAVEFRONT_TOKEN> -n [K8S_CLUSTER_NAME] -v [VERSION]"
  echo "  -c wavefront instance name (default: 'qa4')"
  echo "  -t wavefront token (required)"
  echo "  -n config cluster name for metric grouping (default: \$(whoami)-<default version from file>-release-test)"
  echo "  -v collector docker image version (default: load from 'release/VERSION')"
  exit 1
}

function exit_on_fail() {
  $* # run all arguments as a command
  local exit_code=$?
  if [[ $exit_code != 0 ]]; then
    echo "Command '$@' exited with exit code '$exit_code'"
    exit $exit_code
  fi
}

function main() {
  cd "${SCRIPT_DIR}" # hack/test

  local MAX_QUERY_TIMES=30
  local CURL_WAIT=7
  local NS="wavefront-collector"

  # REQUIRED
  local WAVEFRONT_TOKEN=

  local WF_CLUSTER=qa4
  local VERSION
  VERSION="$(cat "${COLLECTOR_REPO_ROOT}"/release/NEXT_RELEASE_VERSION)"
  local K8S_ENV
  K8S_ENV=$(k8s_env | awk '{print tolower($0)}')
  local K8S_CLUSTER_NAME
  K8S_CLUSTER_NAME=$(whoami)-${K8S_ENV}-$(date +"%y%m%d")

  while getopts ":c:t:n:v:" opt; do
    case $opt in
      c)  WF_CLUSTER="$OPTARG" ;;
      t)  WAVEFRONT_TOKEN="$OPTARG" ;;
      n)  K8S_CLUSTER_NAME="$OPTARG" ;;
      v)  VERSION="$OPTARG" ;;
      \?) print_usage_and_exit "Invalid option: -$OPTARG" ;;
    esac
  done

  if [[ -z ${WAVEFRONT_TOKEN} ]]; then
    print_msg_and_exit "-t <WAVEFRONT_TOKEN> is required"
  fi

  local VERSION_IN_DECIMAL="${VERSION%.*}"
  local VERSION_IN_DECIMAL+="$(echo "${VERSION}" | cut -d '.' -f3)"
  local VERSION_IN_DECIMAL="$(echo "${VERSION_IN_DECIMAL}" | sed 's/0$//')"

  wait_for_cluster_ready "${NS}"

  exit_on_fail wait_for_query_match_exact "${VERSION_IN_DECIMAL}" "at(\"end\", 2m, ts(kubernetes.collector.version, cluster=\"${K8S_CLUSTER_NAME}\" AND installation_method=\"manual\"))"
  exit_on_fail wait_for_query_non_zero "at(\"end\", 2m, ts(kubernetes.cluster.pod.count, cluster=\"${K8S_CLUSTER_NAME}\"))"
  exit_on_fail wait_for_query_non_zero "at(\"end\", 2m, ts(mysql.connections, cluster=\"${K8S_CLUSTER_NAME}\"))"

  # We aren't currently checking for units here (eg. 1250 MiB vs 1.25 GiB), so there's a possibility that the following check could fail
  # We don't believe that is likely to happen due to the size of our environments but we can modify this in the future if that is a problem.
  local NODE_NAME
  NODE_NAME="$(kubectl get nodes -o json | jq -r '.items[] | objects | .metadata.name')"

  while IFS= read -r node; do
    echo "Checking node metrics for: ${node}"

    local EXPECTED_NODE_CPU_REQUEST
    EXPECTED_NODE_CPU_REQUEST="$(kubectl describe node "${node}" | grep -A6 "Allocated resources" | grep "cpu" | awk '{print $2}' | tr -dc '0-9\n')"
    if [[ "${EXPECTED_NODE_CPU_REQUEST}" =~ ^[0-9]$ ]]; then
      EXPECTED_NODE_CPU_REQUEST="${EXPECTED_NODE_CPU_REQUEST}000"
    fi
    exit_on_fail wait_for_query_match_exact "${EXPECTED_NODE_CPU_REQUEST}.000" "at(\"end\", 2m, ts(kubernetes.node.cpu.request, cluster=\"${K8S_CLUSTER_NAME}\"AND source=\"${node}\"))"

    local EXPECTED_NODE_CPU_LIMIT
    EXPECTED_NODE_CPU_LIMIT="$(kubectl describe node "${node}" | grep -A6 "Allocated resources" | grep "cpu" | awk '{print $4}' | tr -dc '0-9\n')"
    if [[ "${EXPECTED_NODE_CPU_LIMIT}" =~ ^[0-9]$ ]]; then
      EXPECTED_NODE_CPU_LIMIT="${EXPECTED_NODE_CPU_LIMIT}000"
    fi
    exit_on_fail wait_for_query_match_exact "${EXPECTED_NODE_CPU_LIMIT}.000" "at(\"end\", 2m, ts(kubernetes.node.cpu.limit, cluster=\"${K8S_CLUSTER_NAME}\"AND source=\"${node}\"))"

    local EXPECTED_NODE_MEMORY_REQUEST
    EXPECTED_NODE_MEMORY_REQUEST="$(kubectl describe node "${node}" | grep -A6 "Allocated resources" | grep "memory" | awk '{print $2}' | numfmt --from=auto)"
    exit_on_fail wait_for_query_match_exact "${EXPECTED_NODE_MEMORY_REQUEST}.000" "at(\"end\", 2m, ts(kubernetes.node.memory.request, cluster=\"${K8S_CLUSTER_NAME}\" AND source=\"${node}\"))"

    local EXPECTED_NODE_MEMORY_LIMIT
    EXPECTED_NODE_MEMORY_LIMIT="$(kubectl describe node "${node}" | grep -A6 "Allocated resources" | grep "memory" | awk '{print $4}' | numfmt --from=auto)"
    exit_on_fail wait_for_query_match_exact "${EXPECTED_NODE_MEMORY_LIMIT}.000" "at(\"end\", 2m, ts(kubernetes.node.memory.limit, cluster=\"${K8S_CLUSTER_NAME}\"AND source=\"${node}\"))"

    local EXPECTED_NODE_EPHERMERAL_STORAGE_REQUEST
    EXPECTED_NODE_EPHERMERAL_STORAGE_REQUEST="$(kubectl describe node "${node}" | grep -A6 "Allocated resources" | grep "ephemeral-storage" | awk '{print $2}' | numfmt --from=auto)"
    exit_on_fail wait_for_query_match_exact "${EXPECTED_NODE_EPHERMERAL_STORAGE_REQUEST}.000" "at(\"end\", 2m, ts(kubernetes.node.ephemeral_storage.request, cluster=\"${K8S_CLUSTER_NAME}\" AND source=\"${node}\"))"

    local EXPECTED_NODE_EPHERMERAL_STORAGE_LIMIT
    EXPECTED_NODE_EPHERMERAL_STORAGE_LIMIT="$(kubectl describe node "${node}" | grep -A6 "Allocated resources" | grep "ephemeral-storage" | awk '{print $4}' | numfmt --from=auto)"
    exit_on_fail wait_for_query_match_exact "${EXPECTED_NODE_EPHERMERAL_STORAGE_LIMIT}.000" "at(\"end\", 2m, ts(kubernetes.node.ephemeral_storage.limit, cluster=\"${K8S_CLUSTER_NAME}\" AND source=\"${node}\"))"
  done <<< "${NODE_NAME}"

  local PROM_EXAMPLE_EXPECTED_COUNT="3"
  exit_on_fail wait_for_query_match_exact "${PROM_EXAMPLE_EXPECTED_COUNT}" "at(\"end\", 2m, ts(prom-example.schedule.activity.decision.counter, cluster=\"${K8S_CLUSTER_NAME}\"))"
}

main $@
