#!/usr/bin/bash -e

export KUBECONFIG=$(mktemp)
sheepctl -n k8po-team lock list -j | jq -r '. | map(select(.status == "locked" and .pool_name != null and (.pool_name | contains("tkg")))) | .[0].access' | jq -r '.tkg[0].kubeconfig' > $KUBECONFIG
chmod go-r $KUBECONFIG
kubectl config current-context
echo make clean-cluster
echo make -C operator integration-test
echo make clean-cluster