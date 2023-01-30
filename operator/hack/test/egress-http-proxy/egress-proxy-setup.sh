#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

OPERATOR_REPO_ROOT=$(git rev-parse --show-toplevel)/operator
source "${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/egress-proxy-setup-functions.sh"

function main() {
  deploy_egress_proxy
  create_mitmproxy-ca-cert_pem_file
  green "HTTP Proxy CAcert:"
  cat "${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem"
}

main "$@"