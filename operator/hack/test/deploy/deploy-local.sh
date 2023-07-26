#!/bin/bash
set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

OPERATOR_REPO_ROOT=${REPO_ROOT}/operator

NS=observability-system

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-c wavefront instance name (default: 'nimba')"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-l wavefront logging token (required)"
  echo -e "\t-n config cluster name for metric grouping (default: \$(whoami)-<default version from file>-release-test)"
  exit 1
}

function main() {

  # REQUIRED
  local WAVEFRONT_TOKEN=
  local WAVEFRONT_LOGGING_TOKEN=
  local WAVEFRONT_URL="https://nimba.wavefront.com"
  local WF_CLUSTER=nimba
  local CONFIG_CLUSTER_NAME
  CONFIG_CLUSTER_NAME=$(create_cluster_name)

  while getopts ":c:t:l:n:p:" opt; do
    case $opt in
    c)
      WF_CLUSTER="$OPTARG"
      ;;
    t)
      WAVEFRONT_TOKEN="$OPTARG"
      ;;
    l)
      WAVEFRONT_LOGGING_TOKEN="$OPTARG"
      ;;
    n)
      CONFIG_CLUSTER_NAME="$OPTARG"
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ -z ${WAVEFRONT_TOKEN} ]]; then
    print_usage_and_exit "wavefront token required"
  fi


  kubectl delete -f ${REPO_ROOT}/deploy/wavefront-operator.yaml || true
  kubectl apply -f ${REPO_ROOT}/deploy/wavefront-operator.yaml
  kubectl create -n ${NS} secret generic wavefront-secret --from-literal token=${WAVEFRONT_TOKEN} || true
  kubectl create -n ${NS} secret generic wavefront-secret-logging --from-literal token=${WAVEFRONT_LOGGING_TOKEN} || true

  cat <<EOF | kubectl apply -f -
  apiVersion: wavefront.com/v1alpha1
  kind: Wavefront
  metadata:
    name: wavefront
    namespace: ${NS}
  spec:
    clusterName: $CONFIG_CLUSTER_NAME
    wavefrontUrl: $WAVEFRONT_URL
    dataCollection:
      metrics:
        enable: true
      logging:
        enable: true
    dataExport:
      wavefrontProxy:
        enable: true
EOF

  wait_for_cluster_ready "$NS"
  kubectl get wavefront -n ${NS}
}

main "$@"
