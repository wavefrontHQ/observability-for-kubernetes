#!/usr/bin/env bash

REPO_ROOT="$(git rev-parse --show-toplevel)"
TKGM_CONTEXT_NAME=tkg-mgmt-vc-admin@tkg-mgmt-vc

function main() {

  mkdir -p /tmp

  KUBECONFIG=/tmp/.kubeconfig

  # You might need to install sheepctl: http://docs.shepherd.run/content/home/installation.html
  sheepctl -n k8po-team lock list -j | jq -r '.[0].access' | jq -r '.tkg[0].kubeconfig' > "${KUBECONFIG}"

  export KUBECONFIG="/tmp/.kubeconfig:${HOME}/.kube/config"

  kubectl config view --flatten > /tmp/combined-kubeconfig.yaml
  mkdir -p "${HOME}/.kube"
  mv /tmp/combined-kubeconfig.yaml "${HOME}/.kube/config"
  chmod go-r "${HOME}/.kube/config"

  export KUBECONFIG="${HOME}/.kube/config"
  kubectl config use-context ${TKGM_CONTEXT_NAME}
  kubectl get nodes
}

main $@
