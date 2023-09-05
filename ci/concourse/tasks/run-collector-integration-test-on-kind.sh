#!/usr/bin/env bash
set -euo pipefail

if ! [ -x "$(command -v kind)" ]; then
  # For AMD64 / x86_64
  [ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
  # For ARM64
  [ $(uname -m) = aarch64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-arm64
  chmod +x ./kind
  sudo mv ./kind /usr/local/bin/kind
fi

echo "Creating fresh kind cluster with version '${KIND_K8S_VERSION}'..."
kind create cluster \
  --image kindest/node:"${KIND_K8S_VERSION}"

cd repo/collector
make integration-test
