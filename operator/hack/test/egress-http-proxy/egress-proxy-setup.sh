#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)/operator
source ${REPO_ROOT}/hack/test/k8s-utils.sh
source ${REPO_ROOT}/hack/test/egress-http-proxy/egress-proxy-setup-functions.sh

function main() {
  deploy_egress_proxy
  create_mitmproxy-ca-cert_pem_file
  green "HTTP Proxy CAcert:"
  cat ${REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem
}

main "$@"