NS=$(kubectl get namespaces | awk '/collector-targets/ {print $1}')
if [ -z ${NS} ]; then exit 0; fi

echo "Uninstalling targets..."

kubectl patch -n collector-targets pod/pod-stuck-in-terminating --type=json -p '[{"op": "remove", "path": "/metadata/finalizers" }]' || true

kubectl delete -f prom-example.yaml &>/dev/null || true
kubectl delete -f exclude-prom-example.yaml &>/dev/null || true
kubectl delete -f cpu-throttled-prom-example.yaml &>/dev/null || true
kubectl delete -f pending-pod-cannot-be-scheduled.yaml &>/dev/null || true
kubectl delete -f pending-pod-image-cannot-be-loaded.yaml &>/dev/null || true
kubectl delete -f running-pod-crash-loop-backoff.yaml &>/dev/null || true
kubectl delete -f running-pod.yaml &>/dev/null || true
kubectl delete -f running-pod-large-init-container.yaml &>/dev/null || true
kubectl delete -f running-pod-small-init-container.yaml &>/dev/null || true
kubectl delete -f pod-stuck-in-terminating.yaml &>/dev/null || true
kubectl delete -f jobs.yaml &>/dev/null || true

helm uninstall memcached-release --namespace collector-targets &>/dev/null || true

helm uninstall mysql-release --namespace collector-targets &>/dev/null || true

kubectl delete --wait=false namespace collector-targets &>/dev/null || true

echo "Targets uninstalled"