#! /bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

SCRIPT_DIR="$(dirname "$0")"

cd "$SCRIPT_DIR"

echo "Deploying targets..."

kubectl delete namespace collector-targets &> /dev/null || true

wait_for_namespace_created collector-targets

wait_for_namespaced_resource_created collector-targets serviceaccount/default

kubectl apply -f prom-example.yaml >/dev/null
kubectl apply -f exclude-prom-example.yaml >/dev/null
kubectl apply -f cpu-throttled-prom-example.yaml >/dev/null
kubectl apply -f pending-pod-cannot-be-scheduled.yaml >/dev/null
kubectl apply -f pending-pod-image-cannot-be-loaded.yaml >/dev/null

kubectl delete -f jobs.yaml &>/dev/null || true
kubectl apply -f jobs.yaml >/dev/null

helm repo add bitnami https://charts.bitnami.com/bitnami &>/dev/null || true
helm upgrade --install memcached-release bitnami/memcached \
--set resources.requests.memory="100Mi",resources.requests.cpu="100m" \
--set persistence.size=200Mi \
--namespace collector-targets >/dev/null

MEMCACHED_RS=$(kubectl get rs -n collector-targets | awk '/memcached-release/ {print $1}')
kubectl autoscale rs -n collector-targets ${MEMCACHED_RS} --min=2 --max=5 --cpu-percent=80

helm upgrade --install mysql-release bitnami/mysql \
--set auth.rootPassword=password123 \
--set primary.persistence.size=200Mi \
--namespace collector-targets >/dev/null

echo "Finished deploying targets"