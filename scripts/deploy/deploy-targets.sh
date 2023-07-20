#!/usr/bin/env bash
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

kubectl apply -f prom-example.yaml >/dev/null
kubectl apply -f exclude-prom-example.yaml >/dev/null
kubectl apply -f cpu-throttled-prom-example.yaml >/dev/null
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
kubectl apply -f non-running-pod-completed.yaml >/dev/null

kubectl delete -f jobs.yaml &>/dev/null || true
kubectl apply -f jobs.yaml >/dev/null
kubectl delete -f cronjobs.yaml &>/dev/null || true
kubectl apply -f cronjobs.yaml >/dev/null

MEMCACHED_CHART_VERSION='6.3.14'

"$REPO_ROOT"/bin/helm repo add bitnami https://charts.bitnami.com/bitnami >/dev/null || true
"$REPO_ROOT"/bin/helm repo update >/dev/null || true
"$REPO_ROOT"/bin/helm upgrade --install memcached-release bitnami/memcached \
--version ${MEMCACHED_CHART_VERSION} \
--set resources.requests.memory="100Mi",resources.requests.cpu="100m" \
--set persistence.size=200Mi \
--namespace collector-targets >/dev/null

MEMCACHED_RS=$(kubectl get rs -n collector-targets | awk '/memcached-release/ {print $1}')
kubectl autoscale rs -n collector-targets ${MEMCACHED_RS} --max=5 --cpu-percent=80

"$REPO_ROOT"/bin/helm upgrade --install mysql-release bitnami/mysql \
--set auth.rootPassword=password123 \
--set primary.persistence.size=500Mi \
--set image.debug=true \
--namespace collector-targets >/dev/null

echo "Finished deploying targets"