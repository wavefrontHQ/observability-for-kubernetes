#!/usr/bin/env bash
set -e

echo "Cleaning up cluster"

CLUSTER_ROLES=$(kubectl get clusterroles | awk '/wavefront-collector|wavefront|wavefront-wavefront-collector|wavefront-wavefront-legacy-install-detection|wavefront-wavefront-logging/ {print $1}')
if [[ ! -z "$CLUSTER_ROLES" ]] ; then
    echo "Found ClusterRoles: ${CLUSTER_ROLES}"
    kubectl delete --wait=false clusterroles ${CLUSTER_ROLES} || true
	  kubectl delete --wait=false clusterrolebindings ${CLUSTER_ROLES} || true
fi

kubectl delete --wait=false deployment/wavefront-proxy &>/dev/null  || true

NS=$(kubectl get namespaces | awk '/wavefront-collector|wavefront|collector-targets|observability-system/ {print $1}')

if [[ ! -z "$NS" ]] ; then
    echo "Found Namespaces: ${NS}"
    kubectl delete --wait=false namespace ${NS} || true
fi

SCC=$(kubectl get scc | awk '/wavefront-controller-manager-scc|wavefront|wavefront-collector-scc|wavefront-proxy-scc|custom-namespace/ {print $1}')
if [[ ! -z "$SCC" ]] ; then
    echo "Found Securitycontextconstraints: ${SCC}"
    kubectl delete --wait=true scc ${SCC} || true
fi


