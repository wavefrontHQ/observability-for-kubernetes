#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

OPERATOR_DIR="${REPO_ROOT}/operator"
source "${OPERATOR_DIR}/hack/test/control-plane/etcd-cert-setup-functions.sh"

function main() {
  deploy_etcd_cert_printer
  create_etcd_cert_files

  green "etcd CA cert:"
  cat "${OPERATOR_DIR}/build/ca.crt"

  green "etcd Server cert:"
  cat "${OPERATOR_DIR}/build/server.crt"

  green "etcd Server key:"
  cat "${OPERATOR_DIR}/build/server.key"
}

main "$@"
