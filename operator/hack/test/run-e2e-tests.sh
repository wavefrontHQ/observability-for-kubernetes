#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

OPERATOR_REPO_ROOT="${REPO_ROOT}/operator"
source "${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/egress-proxy-setup-functions.sh"
source "${OPERATOR_REPO_ROOT}/hack/test/control-plane/etcd-cert-setup-functions.sh"

SCRIPT_DIR="${OPERATOR_REPO_ROOT}/hack/test"

NS=observability-system
NO_CLEANUP=false

function setup_test() {
  local type=$1
  local wf_url="${2:-${WAVEFRONT_URL}}"
  local cluster_name=${CONFIG_CLUSTER_NAME}-$type

  if [[ "$type" == "control-plane" ]]; then
    if [[ "${K8S_ENV}" != "Kind" && "${K8S_ENV}" != "TKGm" ]]; then
      #  if we need a new environment for control plane test, pull this if check to a new function
      echo "Skipping control-plane test as env is not Kind or TKGm"
      return
    fi
  fi

  echo "Deploying Wavefront CR with Cluster Name: $cluster_name ..."

  wait_for_cluster_ready "$NS"

  sed "s/YOUR_CLUSTER_NAME/$cluster_name/g" ${OPERATOR_REPO_ROOT}/hack/test/deploy/scenarios/wavefront-$type.yaml |
    sed "s/YOUR_WAVEFRONT_URL/$wf_url/g" |
    sed "s/YOUR_API_TOKEN/${WAVEFRONT_TOKEN}/g" |
    sed "s/YOUR_NAMESPACE/${NS}/g" >hack/test/_v1alpha1_wavefront_test.yaml

  if [[ "$type" == "with-http-proxy" ]]; then
    deploy_egress_proxy
    create_mitmproxy-ca-cert_pem_file
    echo "---" >> hack/test/_v1alpha1_wavefront_test.yaml
    yq eval '.stringData.tls-root-ca-bundle = "'"$(< ${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem)"'"' "${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/https-proxy-secret.yaml" >> hack/test/_v1alpha1_wavefront_test.yaml
  fi

  if [[ "$type" == "control-plane" ]]; then
    deploy_etcd_cert_printer
    create_etcd_cert_files
    local operator_yaml_content
    operator_yaml_content=$(cat "${OPERATOR_REPO_ROOT}/build/operator/wavefront-operator.yaml")

    echo "---" >> hack/test/_v1alpha1_wavefront_test.yaml
    echo "${operator_yaml_content}" >> hack/test/_v1alpha1_wavefront_test.yaml
    echo "---" >> hack/test/_v1alpha1_wavefront_test.yaml
    yq eval '.stringData.ca_crt = "'"$(< ${OPERATOR_REPO_ROOT}/build/ca.crt)"'"' "${OPERATOR_REPO_ROOT}/hack/test/control-plane/etcd-certs-secret.yaml" \
      | yq eval '.stringData.server_crt = "'"$(< ${OPERATOR_REPO_ROOT}/build/server.crt)"'"' - \
      | yq eval '.stringData.server_key = "'"$(< ${OPERATOR_REPO_ROOT}/build/server.key)"'"' - \
      >> hack/test/_v1alpha1_wavefront_test.yaml
  fi

  kubectl apply -f hack/test/_v1alpha1_wavefront_test.yaml

  wait_for_cluster_ready "$NS"
}

function run_test_wavefront_metrics() {
  local type=$1
  local cluster_name=${CONFIG_CLUSTER_NAME}-$type
  echo "Running test wavefront metrics, cluster_name $cluster_name, version ${VERSION}..."
  ${OPERATOR_REPO_ROOT}/hack/test/test-wavefront-metrics.sh -t "${WAVEFRONT_TOKEN}" -n "${cluster_name}" -e "$type-test.sh" -o "${VERSION}"
}

function run_test_control_plane_metrics() {
  local type='control-plane'
  if [[ "${K8S_ENV}" != "Kind" && "${K8S_ENV}" != "TKGm" ]]; then
    #  if we need a new environment for control plane test, pull this if check to a new function
    echo "Not running control plane metrics tests on env: ${K8S_ENV}"
    return
  fi

  local cluster_name=${CONFIG_CLUSTER_NAME}-$type
  echo "Running test control plane metrics '$type' ..."
  ${OPERATOR_REPO_ROOT}/hack/test/test-wavefront-metrics.sh -t "${WAVEFRONT_TOKEN}" -n "${cluster_name}" -e "$type-test.sh" -o "${VERSION}"
}

function run_health_checks() {
  local type=$1
  local should_be_healthy="${2:-true}"
  printf "Running health checks ..."

  local health_status=
  for _ in {1..120}; do
    health_status=$(kubectl get wavefront -n $NS --request-timeout=10s -o=jsonpath='{.items[0].status.status}') || true
    if [[ "$health_status" == "Healthy" ]]; then
      break
    fi
    printf "."
    sleep 2
  done

  echo " done."

  if [[ "$health_status" != "Healthy" ]]; then
    red "Health status for $type: expected = true, actual = $health_status"
    exit 1
  fi

  proxyLogErrorCount=$(kubectl logs deployment/wavefront-proxy -n $NS | grep " ERROR " | wc -l | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
  if [[ $proxyLogErrorCount -gt 0 ]]; then
    red "Expected proxy log error count of 0, but got $proxyLogErrorCount"
    kubectl logs deployment/wavefront-proxy -n $NS | grep " ERROR " | tail
    exit 1
  fi
}

function run_unhealthy_checks() {
  local type=$1
  echo "Running unhealthy checks ..."

  for _ in {1..20}; do
    health_status=$(kubectl get wavefront -n $NS --request-timeout=10s -o=jsonpath='{.items[0].status.status}') || true
    if [[ "$health_status" == "Unhealthy" ]]; then
      break
    fi
    printf "."
    sleep 1
  done

  if [[ "$health_status" != "Unhealthy" ]]; then
    red "Health status for $type: expected = false, actual = $health_status"
    exit 1
  else
    yellow "Success got expected error: $(kubectl get wavefront -n $NS -o=jsonpath='{.items[0].status.message}')"
  fi
}

function run_proxy_checks() {
  local blocked_input_count=0
  printf "Running proxy checks ..."

  blocked_input_count=$(kubectl logs deployment/wavefront-proxy -n $NS | grep "WF-410: Too many point tags" | wc -l | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
  if [[ $blocked_input_count -gt 0 ]]; then
    red "Expected 'WF-410: Too many point tags' logs received to be zero, but got $blocked_input_count"
    kubectl logs deployment/wavefront-proxy -n $NS | grep "WF-410: Too many point tags" | tail
    exit 1
  fi

  echo " done."
}

function run_k8s_events_checks() {
  local external_event_count=0
  local missing_event_categories_count
  local received_event_categories_count
  local external_events_fail_count
  local external_events_results_file
  external_events_results_file=$(mktemp)

  wait_for_cluster_ready
  sleep 3
  start_forward_test_proxy "observability-system" "test-proxy" /dev/null
  trap 'stop_forward_test_proxy /dev/null' EXIT

  "$REPO_ROOT/scripts/deploy/deploy-event-targets.sh"

  printf "Asserting external events .."
  for i in {1..10}; do
    while true; do # wait until we get a good connection
      RES_CODE=$(curl --silent --output "$external_events_results_file" --write-out "%{http_code}" "http://localhost:8888/events/external/assert" || echo "000")
      if [[ $RES_CODE -ge 200 ]]; then
        break
      fi
    done

    external_event_count=$(jq ".EventCount" "$external_events_results_file")
    if [[ $external_event_count -gt 0 ]]; then
      missing_event_categories_count=$(jq "(.MissingEventCategories | length)" "$external_events_results_file")
      if [[ $missing_event_categories_count -eq 0 ]]; then
        printf " in %d tries" "$i"
        break
      fi
    fi

    printf "."
    sleep 3 # flush interval
  done
  echo " done."

  echo "External events results file: $external_events_results_file"
  # Helpful for debugging:
  # cat "$external_events_results_file" | jq

  if [[ $RES_CODE -ge 400 ]]; then
    red "INVALID EXTERNAL EVENTS"
    exit 1
  fi

  if [[ $external_event_count -eq 0 ]]; then
    red "External events were never received by test-proxy"
    exit 1
  fi

  missing_event_categories_count=$(jq "(.MissingEventCategories | length)" "$external_events_results_file")
  if [[ $missing_event_categories_count -gt 0 ]]; then
    red "FAILED: EXPECTED EXTERNAL EVENTS WERE NOT RECEIVED"
    red "Missing: $missing_event_categories_count"
    red "External event categories missing:"
    jq '.MissingEventCategories' "$external_events_results_file"

    received_event_categories_count=$(jq "(.ReceivedEventCategories | length)" "$external_events_results_file")
    red "Received: $received_event_categories_count"
    if [[ $received_event_categories_count -gt 0 ]]; then
      red "External event categories received:"
      jq '.ReceivedEventCategories | keys' "$external_events_results_file"
    fi

    red "Total: $external_event_count"
    exit 1
  fi

  external_events_fail_count=$(jq "(.BadEventJSONs | length) + (.MissingFields | length) + (.FirstTimestampsMissing | length) + (.LastTimestampsInvalid | length)" "$external_events_results_file")
  if [[ $external_events_fail_count -gt 0 ]]; then
    red "BadEventJSONs: $(jq "(.BadEventJSONs | length)" "$external_events_results_file")"
    red "MissingFields: $(jq "(.MissingFields | length)" "$external_events_results_file")"
    red "FirstTimestampsMissing: $(jq "(.FirstTimestampsMissing | length)" "$external_events_results_file")"
    red "LastTimestampsInvalid: $(jq "(.LastTimestampsInvalid | length)" "$external_events_results_file")"
    jq '{BadEventJSONs, MissingFields, FirstTimestampsMissing, LastTimestampsInvalid}' "$external_events_results_file"
    exit 1
  fi

  yellow "Integration test complete. $external_event_count events were received."

  "$REPO_ROOT/scripts/deploy/uninstall-targets.sh"
  stop_forward_test_proxy /dev/null

  PROXY_NAME="test-proxy"
  METRICS_FILE_DIR="$SCRIPT_DIR/metrics"
  METRICS_FILE_NAME="k8s-events-only"
  source "$REPO_ROOT/scripts/compare-test-proxy-metrics.sh"
}

function run_common_metrics_checks() {
  OPERATOR_TEST=true "$REPO_ROOT/scripts/deploy/deploy-targets.sh"
  wait_for_cluster_ready

  PROXY_NAME="test-proxy"
  METRICS_FILE_DIR="$SCRIPT_DIR/metrics"
  METRICS_FILE_NAME="common-metrics"
  source "$REPO_ROOT/scripts/compare-test-proxy-metrics.sh"
}

function clean_up_test() {
  local type=$1
  echo "Cleaning Up Test '$type' ..."

  kubectl delete -f hack/test/_v1alpha1_wavefront_test.yaml || true

  if [[ "$type" == "with-http-proxy" ]]; then
    delete_egress_proxy
  fi

  if [[ "$type" == "control-plane" ]]; then
    delete_etcd_cert_printer
  fi

  wait_for_proxy_termination "$NS"

  if [[ "$(k8s_env)" == "Kind" ]]; then
    # kill ssh tunnel if it's still open
    if [[ -f /tmp/kind-tunnel-pid ]]; then
       echo "Cleaning Up kind ssh tunnel ..."
       kill -9 $(cat /tmp/kind-tunnel-pid) || true
       rm /tmp/kind-tunnel-pid || true
    fi
  fi
}

function clean_up_port_forward() {
  if [[ -f "$PF_OUT" ]]; then
    echo "PF_OUT:"
    cat "$PF_OUT"
  fi

  if [[ -f "$CURL_OUT" ]]; then
    echo "CURL_OUT:"
    jq '.' "$CURL_OUT"
  fi

  if [[ -f "$CURL_ERR" ]]; then
    echo "CURL_ERR:"
    cat "$CURL_ERR"
  fi

  if [[ -n "$(jobs -l)" ]]; then
    echo "Killing jobs: $(jobs -l)"
    kill $(jobs -p) &>/dev/null || true
  fi
}

function checks_to_remove() {
  local file_name=$1
  local component_name=$2
  local checks=$3
  local tempFile

  for i in ${checks//,/ }; do
    tempFile=$(mktemp)
    local excludeCheck
    excludeCheck=$(echo $i | sed -r 's/:/ /g')
    local awk_command="!/.*$excludeCheck.*$component_name|.*$component_name.*$excludeCheck/"
    cat "$file_name" | awk "$awk_command" >"$tempFile" && mv "$tempFile" "$file_name"
  done
}

function run_static_analysis() {
  local type=$1
  local k8s_env
  k8s_env=$(k8s_env)
  echo "Running static analysis ..."

  local resources_yaml_file
  resources_yaml_file=$(mktemp)
  local exit_status=0
  kubectl get "$(kubectl api-resources --verbs=list --namespaced -o name | tr '\n' ',' | sed s/,\$//)" --ignore-not-found -n $NS -o yaml |
    yq '.items[] | split_doc' - >"$resources_yaml_file"

  echo "Running static analysis: kube-linter"

  local kube_lint_results_file
  kube_lint_results_file=$(mktemp)
  local kube_lint_check_errors
  kube_lint_check_errors=$(mktemp)
  kube-linter lint "$resources_yaml_file" --format json 1>"$kube_lint_results_file" 2>/dev/null || true

  local current_lint_errors
  current_lint_errors="$(jq '.Reports | length' "$kube_lint_results_file")"
  yellow "Kube linter error count: ${current_lint_errors}"

  jq -r '.Reports[] | "|" + .Check + "|  " +.Object.K8sObject.GroupVersionKind.Kind + " " + .Object.K8sObject.Namespace + "/" +  .Object.K8sObject.Name + ": " + .Diagnostic.Message' "$kube_lint_results_file" 1>"$kube_lint_check_errors" 2>/dev/null || true

  #REMOVE KNOWN CHECKS
  #non root checks for logging
  checks_to_remove "$kube_lint_check_errors" "wavefront-logging" "run-as-non-root,no-read-only-root-fs"
  #sensitive-host-mounts checks for the collector
  checks_to_remove "$kube_lint_check_errors" "collector" "sensitive-host-mounts"
  #non root checks for the controller manager to support openshift
  checks_to_remove "$kube_lint_check_errors" "wavefront-controller-manager" "run-as-non-root"

  current_lint_errors=$(cat "$kube_lint_check_errors" | wc -l)
  yellow "Kube linter error count (with known errors removed): ${current_lint_errors}"
  local known_lint_errors=0
  if [ $current_lint_errors -gt $known_lint_errors ]; then
    red "Failure: Expected error count = $known_lint_errors"
    cat "$kube_lint_check_errors"
    exit_status=1
  fi

  echo "Running static analysis: kube-score"
  local kube_score_results_file
  kube_score_results_file=$(mktemp)
  local kube_score_critical_errors
  kube_score_critical_errors=$(mktemp)
  kube-score score "$resources_yaml_file" --ignore-test pod-networkpolicy --output-format ci >"$kube_score_results_file" || true

  grep '\[CRITICAL\]' "$kube_score_results_file" >"$kube_score_critical_errors"
  local current_score_errors
  current_score_errors=$(cat "$kube_score_critical_errors" | wc -l)
  yellow "Kube score error count: ${current_score_errors}"

  #REMOVE KNOWN CHECKS
  #non root checks for logging
  checks_to_remove "$kube_score_critical_errors" "wavefront-logging" "security:context,low:user:ID,low:group:ID"
  if [[ "$k8s_env" == "Kind" ]]; then
    checks_to_remove "$kube_score_critical_errors" "wavefront-controller-manager" "ImagePullPolicy"
  fi
  #non root checks for the controller manager to support openshift
  checks_to_remove "$kube_score_critical_errors" "wavefront-controller-manager" "security:context,low:user:ID,low:group:ID"

  current_score_errors=$(cat "$kube_score_critical_errors" | wc -l)
  yellow "Kube score error count (with known errors removed): ${current_score_errors}"
  local known_score_errors=0
  if [ $current_score_errors -gt $known_score_errors ]; then
    red "Failure: Expected error count = $known_score_errors"
    cat "$kube_score_critical_errors"
    exit_status=1
  fi

  echo "Running static analysis: ServiceAccount automountServiceAccountToken checks"
  local automountToken=
  local service_accounts
  service_accounts=$(kubectl get serviceaccounts -l app.kubernetes.io/name=wavefront -n $NS -o name | tr '\n' ',' | sed "s/serviceaccount\///g" | sed s/,\$//)

  for i in ${service_accounts//,/ }; do
    automountToken=$(kubectl get serviceaccount $i -n $NS -o=jsonpath='{.automountServiceAccountToken}' | tr -d '\n')
    if [[ $automountToken != "false" ]]; then
      red "Failure: Expected automountToken in $i to be \"false\", but was $automountToken"
      exit 1
    fi
  done

  echo "Running static analysis: Pod automountServiceAccountToken checks"
  local pods
  pods=$(kubectl get pods -l app.kubernetes.io/name=wavefront -n $NS -o name | tr '\n' ',' | sed "s/pod\///g" | sed s/,\$//)

  for i in ${pods//,/ }; do
    automountToken=$(kubectl get pod $i -n $NS -o=jsonpath='{.spec.automountServiceAccountToken}' | tr -d '\n')
    if [[ $automountToken == "" ]]; then
      red "Failure: Expected automountToken in $i to be set"
      exit 1
    fi
  done

  if [[ $exit_status -ne 0 ]]; then
    exit $exit_status
  fi
}

function run_logging_checks() {
  printf "Running logging checks ..."
  local max_logs_received=0
  for _ in {1..12}; do
    max_logs_received=$(kubectl -n $NS logs -l app.kubernetes.io/name=wavefront -l app.kubernetes.io/component=proxy --tail=-1 | grep "Logs received" | awk 'match($0, /[0-9]+ logs\/s/) { print substr( $0, RSTART, RLENGTH )}' | awk '{print $1}' | sort -n | tail -n1 2>/dev/null)
    if [[ $max_logs_received -gt 0 ]]; then
      break
    fi
    sleep 5
  done

  if [[ $max_logs_received -eq 0 ]]; then
    red "Expected max logs received to be greater than 0, but got $max_logs_received"
    exit 1
  fi
  echo " done."
}

function run_logging_integration_checks() {
  echo "Running logging checks with test-proxy ..."

  # send request to the fake proxy control endpoint and check status code for success
  CURL_OUT=$(mktemp)
  CURL_ERR=$(mktemp)
  PF_OUT=$(mktemp)

  echo "first jobs..."
  start_forward_test_proxy "observability-system" "test-proxy" "$PF_OUT"
  trap 'stop_forward_test_proxy "/dev/null"' EXIT
  trap 'clean_up_port_forward' EXIT

  for _ in {1..60}; do
    CURL_CODE=$(curl --silent --show-error --output "$CURL_OUT" --stderr "$CURL_ERR" --write-out "%{http_code}" "http://localhost:8888/logs/assert" || echo "000")
    if [[ $CURL_CODE -eq 200 ]]; then
      break
    fi
    sleep 1
  done

  # Helpful for debugging:
  # cat "${CURL_OUT}" >/tmp/test

  if [[ $CURL_CODE -eq 204 ]]; then
    red "Logs were never received by test proxy"
    kubectl -n observability-system exec deployment/test-proxy -- cat /logs/test-proxy.log
    exit 1
  fi

  # TODO look at result and pass or fail test
  if [[ $CURL_CODE -gt 399 ]]; then
    red "LOGGING ASSERTION FAILURE"
    kubectl -n observability-system exec deployment/test-proxy -- cat /logs/test-proxy.log
    exit 1
  fi

  hasValidFormat=$(jq -r .hasValidFormat "${CURL_OUT}")
  if [[ ${hasValidFormat} -ne 1 ]]; then
    red "Test proxy received logs with invalid format"
    kubectl -n observability-system exec deployment/test-proxy -- cat /logs/test-proxy.log
    exit 1
  fi

  hasValidTags=$(jq -r .hasValidTags "${CURL_OUT}")
  missingExpectedTags="$(jq .missingExpectedTags "${CURL_OUT}")"
  missingExpectedTagsCount="$(jq .missingExpectedTagsCount "${CURL_OUT}")"

  missingExpectedOptionalTags="$(jq .missingExpectedOptionalTagsMap "${CURL_OUT}")"

  emptyExpectedTags="$(jq .emptyExpectedTags "${CURL_OUT}")"
  emptyExpectedTagsCount="$(jq .emptyExpectedTagsCount "${CURL_OUT}")"

  unexpectedAllowedLogs="$(jq .unexpectedAllowedLogs "${CURL_OUT}")"
  unexpectedAllowedLogsCount="$(jq .unexpectedAllowedLogsCount "${CURL_OUT}")"

  unexpectedDeniedTags="$(jq .unexpectedDeniedTags "${CURL_OUT}")"
  unexpectedDeniedTagsCount="$(jq .unexpectedDeniedTagsCount "${CURL_OUT}")"

  receivedLogCount=$(jq .receivedLogCount "${CURL_OUT}")

  if [[ ${hasValidTags} -ne 1 ]]; then
    red "Invalid tags were found:"

    if [[ ${missingExpectedOptionalTags} != "null" ]]; then
      echo ""
      red "* Test proxy did not receive expected optional tags:"
      red "${missingExpectedOptionalTags}"
    fi

    if [[ ${missingExpectedTags} != "null" ]]; then
      echo ""
      red "* Test proxy received logs (${missingExpectedTagsCount}/${receivedLogCount} logs) that were missing expected tags:"
      red "${missingExpectedTags}"
    fi

    if [[ ${emptyExpectedTags} != "null" ]]; then
      echo ""
      red "* Test proxy received logs (${emptyExpectedTagsCount}/${receivedLogCount} logs) with expected tags that were empty:"
      red "${emptyExpectedTags}"
    fi

    if [[ ${unexpectedAllowedLogs} != "null" ]]; then
      echo ""
      red "* Test proxy received (${unexpectedAllowedLogsCount}/${receivedLogCount} logs) logs that should not have been there because none of their tags were in the allowlist:"
      red "${unexpectedAllowedLogs}"
    fi

    if [[ ${unexpectedDeniedTags} != "null" ]]; then
      echo ""
      red "* Test proxy received (${unexpectedDeniedTagsCount}/${receivedLogCount} logs) logs that should not have been there because some of their tags were in the denylist:"
      red "${unexpectedDeniedTags}"
    fi

    exit 1
  fi

  rm -f "$CURL_OUT" "$CURL_ERR" "$PF_OUT" || true
  stop_forward_test_proxy "/dev/null"

  yellow "Integration test complete. ${receivedLogCount} logs were checked."
}

function run_test() {
  local type=$1
  shift
  local checks=("$@")
  echo ""
  green "[$(date +%H:%M:%S)] Running test $type"

  setup_test $type

  if [[ " ${checks[*]} " =~ " unhealthy " ]]; then
    run_unhealthy_checks $type
  elif [[ " ${checks[*]} " =~ " health " ]]; then
    run_health_checks $type
  fi

  if [[ " ${checks[*]} " =~ " static_analysis " ]]; then
    run_static_analysis $type
  fi

  if [[ " ${checks[*]} " =~ " test_wavefront_metrics " ]]; then
    run_test_wavefront_metrics $type
  fi

  if [[ " ${checks[*]} " =~ " test_control_plane_metrics " ]]; then
    run_test_control_plane_metrics
  fi

  if [[ " ${checks[*]} " =~ " logging " ]]; then
    run_logging_checks
  fi

  if [[ " ${checks[*]} " =~ " proxy " ]]; then
    run_proxy_checks
  fi

  if [[ " ${checks[*]} " =~ " logging-integration-checks " ]]; then
    run_logging_integration_checks
  fi

  if [[ " ${checks[*]} " =~ " k8s_events " ]]; then
    run_k8s_events_checks
  fi

  if [[ " ${checks[*]} " =~ " common-metrics-check " ]]; then
    run_common_metrics_checks
  fi

	if ! $NO_CLEANUP; then
		clean_up_test $type
	fi

  green "[$(date +%H:%M:%S)] Successfully ran $type test!"
}

function run_test_if_enabled() {
  local enabled_test=$1
  local type=$2
  shift
  shift
  local checks=("$@")

  if [[ $type == $enabled_test ]]; then
    run_test $type $checks
  fi
}

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-c wavefront instance name (default: 'qa4')"
  echo -e "\t-v operator version (default: load from 'release/OPERATOR_VERSION')"
  echo -e "\t-n k8s cluster name (default: '$(create_cluster_name)')"
  echo -e "\t-r tests to run (runs all by default)"
  echo -e "\t-d namespace to create CR in (default: observability-system)"
  echo -e "\t-e no cl[e]anup after test to debug testing framework"
  echo -e "\t-y operator YAML content command (default: '${OPERATOR_REPO_ROOT}/build/operator/wavefront-operator.yaml')"
  echo -e "\t\t example usage: -y 'curl https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/rc/operator/wavefront-operator-PR-116.yaml'"
  exit 1
}

function main() {
  # REQUIRED
  local WAVEFRONT_TOKEN=

  local WAVEFRONT_URL="https:\/\/qa4.wavefront.com"
  local WF_CLUSTER=qa4
  local VERSION
  VERSION=$(get_next_operator_version)
  local K8S_ENV
  K8S_ENV=$(k8s_env)
  local CONFIG_CLUSTER_NAME
  CONFIG_CLUSTER_NAME=$(create_cluster_name)
  local tests_to_run=()

  while getopts ":t:c:v:n:r:d:e" opt; do
    case $opt in
    t)
      WAVEFRONT_TOKEN="$OPTARG"
      ;;
    c)
      WF_CLUSTER="$OPTARG"
      ;;
    v)
      VERSION="$OPTARG"
      ;;
    n)
      CONFIG_CLUSTER_NAME="$OPTARG"
      ;;
    r)
      tests_to_run+=("$OPTARG")
      ;;
    d)
      NS="$OPTARG"
      ;;
		e)
			NO_CLEANUP=true
			;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ ${#tests_to_run[@]} -eq 0 ]]; then
    tests_to_run=(
      "validation-errors"
      "validation-legacy"
      "validation-errors-preprocessor-rules"
      "allow-legacy-install"
      "basic"
      "advanced"
      "logging-integration"
      "with-http-proxy"
      "k8s-events-only"
      "control-plane"
      "common-metrics"
    )
  fi

  if [[ -z ${WAVEFRONT_TOKEN} ]]; then
    print_usage_and_exit "wavefront token required"
  fi

  if [[ -z ${CONFIG_CLUSTER_NAME} ]]; then
    CONFIG_CLUSTER_NAME=$(create_cluster_name)
  fi

  cd "$OPERATOR_REPO_ROOT"

  for test_to_run in ${tests_to_run[*]} ; do
    run_test_if_enabled $test_to_run "k8s-events-only" "k8s_events"
    run_test_if_enabled $test_to_run "validation-errors" "unhealthy"
    run_test_if_enabled $test_to_run "validation-legacy" "unhealthy"
    run_test_if_enabled $test_to_run "validation-errors-preprocessor-rules" "unhealthy"
    run_test_if_enabled $test_to_run "logging-integration" "logging-integration-checks"
    run_test_if_enabled $test_to_run "allow-legacy-install" "healthy"
    run_test_if_enabled $test_to_run "basic" "health" "static_analysis" "test_wavefront_metrics" "proxy"
    run_test_if_enabled $test_to_run "common-metrics" "common-metrics-check"
    run_test_if_enabled $test_to_run "advanced" "health" "test_wavefront_metrics" "logging" "proxy"
    run_test_if_enabled $test_to_run "with-http-proxy" "health" "test_wavefront_metrics"
    run_test_if_enabled $test_to_run "control-plane" "test_control_plane_metrics"
  done

  exit 0

}

main "$@"
