#!/bin/bash

if [ -z "${KIND_K8S_VERSION}" ]; then
  echo 'Usage: KIND_K8S_VERSION cannot be empty'
  exit 1
fi

if ! kubectl config use-context "kind-kind" 2>/dev/null ; then
  echo "Creating fresh kind cluster with version '${KIND_K8S_VERSION}'..."
  kind delete cluster
  kind create cluster \
    --image kindest/node:"${KIND_K8S_VERSION}"
fi
