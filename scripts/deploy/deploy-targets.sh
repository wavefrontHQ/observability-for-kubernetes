#!/usr/bin/env bash
if [ -z "${OPERATOR_TEST}" ]; then
  OPERATOR_TEST='false'
fi

set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

SCRIPT_DIR="$(dirname "$0")"

cd "$SCRIPT_DIR"

echo "Deploying targets..."

kubectl patch -n collector-targets pod/pod-stuck-in-terminating --type=json -p '[{"op": "remove", "path": "/metadata/finalizers" }]' &>/dev/null || true

kubectl delete --ignore-not-found=true namespace collector-targets &> /dev/null || true

wait_for_namespace_created collector-targets

wait_for_namespaced_resource_created collector-targets serviceaccount/default

if [ "${OPERATOR_TEST}" != "true" ]; then
  kubectl apply -f prom-example.yaml >/dev/null
  kubectl apply -f exclude-prom-example.yaml >/dev/null
  kubectl apply -f cpu-throttled-prom-example.yaml >/dev/null
else
  kubectl apply -f light-prom-example.yaml
fi

kubectl apply -f pending-pod-cannot-be-scheduled.yaml >/dev/null
kubectl apply -f pending-pod-image-cannot-be-loaded.yaml >/dev/null
kubectl apply -f running-pod-crash-loop-backoff.yaml >/dev/null
kubectl apply -f running-pod.yaml >/dev/null
kubectl apply -f running-pod-large-init-container.yaml >/dev/null
kubectl apply -f running-pod-small-init-container.yaml >/dev/null
kubectl apply -f pod-stuck-in-terminating.yaml >/dev/null
kubectl delete -f pod-stuck-in-terminating.yaml >/dev/null &
kubectl apply -f workload-not-ready.yaml >/dev/null
kubectl apply -f replicaset-no-owner.yaml >/dev/null

kubectl delete -f jobs.yaml &>/dev/null || true
kubectl apply -f jobs.yaml >/dev/null

MEMCACHED_CHART_VERSION='6.3.14'

helm repo add bitnami https://charts.bitnami.com/bitnami >/dev/null || true
helm repo update >/dev/null || true
helm upgrade --install memcached-release bitnami/memcached \
--version ${MEMCACHED_CHART_VERSION} \
--set resources.requests.memory="100Mi",resources.requests.cpu="100m" \
--set persistence.size=200Mi \
--namespace collector-targets >/dev/null
# Note on requests and limits: limits are automatically created on pod containers, but not pods

MEMCACHED_RS=$(kubectl get rs -n collector-targets | awk '/memcached-release/ {print $1}')
kubectl autoscale rs -n collector-targets ${MEMCACHED_RS} --max=5 --cpu-percent=80

if [ "${OPERATOR_TEST}" != "true" ]; then
  helm upgrade --install mysql-release bitnami/mysql \
  --set auth.rootPassword=password123 \
  --set primary.persistence.size=500Mi \
  --set image.debug=true \
  --namespace collector-targets >/dev/null
fi

echo "Finished deploying targets"