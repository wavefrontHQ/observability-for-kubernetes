#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

function delete_cluster_roles() {
  local cluster_roles=''

  cluster_roles=$(kubectl get clusterroles | awk '/wavefront/ {print $1}')
  cluster_role_bindings=$(kubectl get clusterrolebindings | awk '/wavefront/ {print $1}')

  if [[ -n "${cluster_roles}" ]]; then
    kubectl delete --ignore-not-found=true --wait=false clusterroles ${cluster_roles} || true
  fi

  if [[ -n "${cluster_role_bindings}" ]]; then
    kubectl delete --ignore-not-found=true --wait=false clusterrolebindings ${cluster_role_bindings} || true
  fi
}

function delete_namespaces() {
  local ns=''

  ns=$(kubectl get namespaces | awk '/wavefront|collector-targets|observability-system|custom-namespace|tanzu-observability-saas/ {print $1}')

  if [[ -n "${ns}" ]]; then
    kubectl patch -n collector-targets pod/pod-stuck-in-terminating -p '{"metadata":{"finalizers":null}}' &>/dev/null || true
    kubectl delete --ignore-not-found=true --wait=false namespace ${ns} || true
  fi
}

function delete_crd() {
    kubectl delete --ignore-not-found=true --wait=false crd wavefronts.wavefront.com || true
}

function delete_security_context_constraints() {
  local scc=''

  scc=$(kubectl get scc | awk '/wavefront|custom-namespace/ {print $1}')

  if [[ -n "${scc}" ]]; then
    kubectl delete --ignore-not-found=true --wait=true scc ${scc} || true
  fi
}

function main() {
  echo "Cleaning up cluster ..."
  local WAIT_FOR_CLUSTER_TO_FINISH=true

  while getopts ":n" opt; do
    case $opt in
    n)
      WAIT_FOR_CLUSTER_TO_FINISH=false
      ;;
    \?)
      print_usage_and_exit "Invalid option: -$OPTARG"
      ;;
    esac
  done

  kubectl delete --ignore-not-found=true --wait=false deployment/wavefront-proxy || true

  delete_cluster_roles
  delete_namespaces
  delete_crd

  if kubectl get scc &>/dev/null; then
    delete_security_context_constraints
  fi

  if [[ "${WAIT_FOR_CLUSTER_TO_FINISH}" == "true" ]]; then
    wait_for_proxy_termination
    wait_for_cluster_ready
  fi
}

main "$@"
