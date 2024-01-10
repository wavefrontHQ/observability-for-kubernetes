#!/bin/bash
set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 [flags] [options]"
  echo -e "\t-u wavefront instance url (default: 'https://qa4.wavefront.com')"
  echo -e "\t-t wavefront token (required)"
  echo -e "\t-n k8s cluster name (default: '$(create_cluster_name)')"
  exit 1
}

function main() {
  # Required
  local WAVEFRONT_TOKEN=

  # Default
  local NS=observability-system
  local WAVEFRONT_URL='https://qa4.wavefront.com'
  local CONFIG_CLUSTER_NAME
  local INCLUDE_CR_DEPLOYMENT=true
  CONFIG_CLUSTER_NAME=$(create_cluster_name)

  while getopts ":u:t:n:x" opt; do
    case $opt in
    u)
      WAVEFRONT_URL="$OPTARG"
      ;;
    t)
      WAVEFRONT_TOKEN="$OPTARG"
      ;;
    n)
      CONFIG_CLUSTER_NAME="$OPTARG"
      ;;
    x)
      INCLUDE_CR_DEPLOYMENT=false
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  if [[ -z ${WAVEFRONT_TOKEN} ]]; then
    print_usage_and_exit "wavefront token required"
  fi

  kubectl delete -f ${REPO_ROOT}/deploy/wavefront-operator.yaml 2>/dev/null || true
  kubectl apply -f ${REPO_ROOT}/deploy/wavefront-operator.yaml
  kubectl create -n ${NS} secret generic wavefront-secret --from-literal token=${WAVEFRONT_TOKEN} || true

  if [[ "${INCLUDE_CR_DEPLOYMENT}" == "true" ]]; then
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
  fi
}

main "$@"
