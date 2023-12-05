#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

COLLECTOR_REPO_ROOT=$(git rev-parse --show-toplevel)/collector
SCRIPT_DIR="${COLLECTOR_REPO_ROOT}/hack/test"
METRICS_FILE_DIR="${SCRIPT_DIR}/files"
NS=wavefront-collector

function run_fake_proxy_test() {
  local METRICS_FILE_NAME=$1
  local COLLECTOR_YAML=$2
  local EXPERIMENTAL_FEATURES=$3
  local COLLECTOR_CONFIG_YAML=$4

  local USE_TEST_PROXY="true"
  local PROXY_NAME="wavefront-proxy"

  wait_for_cluster_resource_deleted namespace/$NS

  kubectl create namespace $NS
  kubectl apply -f <(sed "s/wavefront-collector/$NS/g" "$REPO_ROOT/scripts/deploy/mysql-config.yaml")
  kubectl apply -f <(sed "s/wavefront-collector/$NS/g" "$REPO_ROOT/scripts/deploy/memcached-config.yaml")

  local additional_args=""
  if [[ -n "${COLLECTOR_YAML:-}" ]]; then
    additional_args="$additional_args -y $COLLECTOR_YAML"
  fi
  if [[ -n "${EXPERIMENTAL_FEATURES:-}" ]]; then
    additional_args="$additional_args -e $EXPERIMENTAL_FEATURES"
  fi
  if [[ -n "${COLLECTOR_CONFIG_YAML:-}" ]]; then
    additional_args="$additional_args -z $COLLECTOR_CONFIG_YAML"
  fi

  "${SCRIPT_DIR}"/deploy.sh \
      -c "$WAVEFRONT_CLUSTER" \
      -t "$WAVEFRONT_TOKEN" \
      -k "$K8S_ENV" \
      -v "$VERSION" \
      -n "$K8S_CLUSTER_NAME" \
      -p "$USE_TEST_PROXY" \
      $additional_args

  wait_for_cluster_ready

  source "${REPO_ROOT}/scripts/compare-test-proxy-metrics.sh"
}

function run_real_proxy_metrics_test() {
	run_real_proxy
  "${SCRIPT_DIR}"/test-wavefront-metrics.sh -t "$WAVEFRONT_TOKEN" -v $(get_next_collector_version)
  green "SUCCEEDED"
}

function run_real_proxy() {
  local USE_TEST_PROXY="false"
  local additional_args="-p $USE_TEST_PROXY"
  if [[ -n "${EXPERIMENTAL_FEATURES:-}" ]]; then
    additional_args="$additional_args -e $EXPERIMENTAL_FEATURES"
  fi


  wait_for_cluster_resource_deleted namespace/$NS
  wait_for_cluster_ready

  "${SCRIPT_DIR}"/deploy.sh \
      -c "$WAVEFRONT_CLUSTER" \
      -t "$WAVEFRONT_TOKEN" \
      -k "$K8S_ENV" \
      -v "$VERSION" \
      -n "$K8S_CLUSTER_NAME" \
      $additional_args
}

function print_usage_and_exit() {
  red "Failure: $1"
  echo "Usage: $0 -c <WAVEFRONT_CLUSTER> -t <WAVEFRONT_TOKEN> -v [VERSION] -r [INTEGRATION_TEST_ARGS...]"
  echo "  -c wavefront instance name (required)"
  echo "  -t wavefront token (required)"
  echo "  -v collector docker image version (default: load from 'release/VERSION')"
  echo "  -k K8s ENV"
  echo "  -n K8s Cluster name"
  echo "  -r tests to run"
  exit 1
}

function check_required_argument() {
  local required_arg=$1
  local failure_msg=$2
  if [[ -z ${required_arg} ]]; then
    print_usage_and_exit "$failure_msg"
  fi
}

function main() {
  local EXIT_CODE=0

  # REQUIRED
  local WAVEFRONT_CLUSTER=
  local WAVEFRONT_TOKEN=
  local VERSION=

  # OPTIONAL/DEFAULT
  local K8S_ENV
  K8S_ENV=$(k8s_env | awk '{print tolower($0)}')
  local K8S_CLUSTER_NAME
  K8S_CLUSTER_NAME=$(whoami)-${K8S_ENV}-$(date +"%y%m%d")
  local EXPERIMENTAL_FEATURES=
  local tests_to_run=()

  while getopts ":c:t:v:k:n:r:" opt; do
    case $opt in
      c)  WAVEFRONT_CLUSTER="$OPTARG" ;;
      t)  WAVEFRONT_TOKEN="$OPTARG" ;;
      v)  VERSION="$OPTARG" ;;
      k)  K8S_ENV="$OPTARG" ;;
      n)  K8S_CLUSTER_NAME="$OPTARG" ;;
      r)  tests_to_run+=("$OPTARG") ;;
      \?) print_usage_and_exit "Invalid option: -$OPTARG" ;;
    esac
  done

  check_required_argument "$WAVEFRONT_CLUSTER" "-c <WAVEFRONT_CLUSTER> is required"
  check_required_argument "$WAVEFRONT_TOKEN" "-t <WAVEFRONT_TOKEN> is required"
  check_required_argument "$VERSION" "-t <VERSION> is required"

  if [[ ${#tests_to_run[@]} -eq 0 ]]; then
    tests_to_run=( "default" )
  fi

  if [[ "${tests_to_run[*]}" =~ "cluster-metrics-only" ]]; then
    green "\n==================== Running fake_proxy cluster-metrics-only test ===================="
    run_fake_proxy_test "cluster-metrics-only" "base/deploy/collector-deployments/5-collector-cluster-metrics-only.yaml"
    ${SCRIPT_DIR}/clean-deploy.sh
  fi
  if [[ "${tests_to_run[*]}" =~ "node-metrics-only" ]]; then
    green "\n==================== Running fake_proxy node-metrics-only test ===================="
    run_fake_proxy_test "node-metrics-only" "base/deploy/collector-deployments/5-collector-node-metrics-only.yaml"
    ${SCRIPT_DIR}/clean-deploy.sh
  fi
  if [[ "${tests_to_run[*]}" =~ "combined" ]]; then
    green "\n==================== Running fake_proxy combined test ===================="
    run_fake_proxy_test "all-metrics" "base/deploy/collector-deployments/5-collector-combined.yaml"
    ${SCRIPT_DIR}/clean-deploy.sh
  fi
  if [[ "${tests_to_run[*]}" =~ "single-deployment" ]]; then
    green "\n==================== Running fake_proxy single-deployment test ===================="
    run_fake_proxy_test "all-metrics" "base/deploy/collector-deployments/5-collector-single-deployment.yaml"
    ${SCRIPT_DIR}/clean-deploy.sh
  fi
  if [[ "${tests_to_run[*]}" =~ "default" ]]; then
    green "\n==================== Running fake_proxy default test ===================="
    run_fake_proxy_test "all-metrics" "${COLLECTOR_REPO_ROOT}/hack/deploy/kubernetes/5-collector-daemonset.yaml"
    ${SCRIPT_DIR}/clean-deploy.sh
  fi
  if [[ "${tests_to_run[*]}" =~ "real-proxy-metrics" ]]; then
    green "\n==================== Running real-proxy-metrics test ===================="
    run_real_proxy_metrics_test
    ${SCRIPT_DIR}/clean-deploy.sh
  fi
  if [[ " ${tests_to_run[*]} " =~ " real-proxy " ]]; then
    green "\n==================== Starting real proxy ===================="
    run_real_proxy
  fi
}

main $@
