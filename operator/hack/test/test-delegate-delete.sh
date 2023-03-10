NS=observability-system

kubectl delete deployment/wavefront-controller-manager -n $NS

kubectl wait --for=delete pod --all --selector="app.kubernetes.io/name=wavefront"  --namespace=$NS --timeout=60s

STATUS="$(kubectl get pods -n observability-system 2>&1)"

if [ "${STATUS}" == "No resources found in ${NS} namespace." ]; then
	echo "Success"
	exit 0
else
  echo "Failed to delegate delete"
  exit 1
fi

