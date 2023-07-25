#!/usr/bin/env bash
set -e

function delete_cluster_roles() {
  local cluster_roles=''

  cluster_roles=$(kubectl get clusterroles | awk '/wavefront/ {print $1}')

  if [[ -n "${cluster_roles}" ]]; then
    echo -e "Found ClusterRoles:\n${cluster_roles}"
    kubectl delete --ignore-not-found=true --wait=false clusterroles ${cluster_roles} || true
    kubectl delete --ignore-not-found=true --wait=false clusterrolebindings ${cluster_roles} || true
  fi
}

function delete_namespaces() {
  local ns=''

  ns=$(kubectl get namespaces | awk '/wavefront|collector-targets|observability-system|custom-namespace|tanzu-observability-saas/ {print $1}')

  if [[ -n "${ns}" ]]; then
    echo -e "Found Namespaces:\n${ns}"
    kubectl patch -n collector-targets pod/pod-stuck-in-terminating -p '{"metadata":{"finalizers":null}}' &>/dev/null || true
    kubectl delete --ignore-not-found=true --wait=false namespace ${ns} || true
  fi
}

function delete_security_context_constraints() {
  local scc=''

  scc=$(kubectl get scc | awk '/wavefront|custom-namespace/ {print $1}')

  if [[ -n "${scc}" ]]; then
    echo -e "Found SecurityContextConstraints:\n${scc}"
    kubectl delete --ignore-not-found=true --wait=true scc ${scc} || true
  fi
}

function main() {
  echo "Cleaning up cluster ..."

  kubectl delete --ignore-not-found=true --wait=false deployment/wavefront-proxy || true

  delete_cluster_roles
  delete_namespaces

  if kubectl get scc &>/dev/null; then
    delete_security_context_constraints
  fi
}

main "$@"
