#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)/operator
source ${REPO_ROOT}/hack/test/k8s-utils.sh

function deploy_egress_proxy() {
  # create the namespace observability-system
  kubectl create namespace observability-system || true

  # deploy the mitmproxy
  kubectl apply -f ${REPO_ROOT}/hack/test/egress-http-proxy/egress-proxy.yaml

  # wait for egress proxy
  wait_for_cluster_ready

  sleep 1
}

function delete_egress_proxy() {
  kubectl delete -f ${REPO_ROOT}/hack/test/egress-http-proxy/egress-proxy.yaml
}

function create_mitmproxy-ca-cert_pem_file() {
  #get httpproxy ip
  export MITM_IP="$(kubectl -n observability-system get services --selector=app=egress-proxy -o jsonpath='{.items[*].spec.clusterIP}')"
  green "HTTP Proxy URL:"
  echo "http://${MITM_IP}:8080"

  # get the ca cert for the mitmproxy
  export MITM_POD="$(kubectl -n observability-system get pods --selector=app=egress-proxy -o jsonpath='{.items[*].metadata.name}')"
  kubectl cp observability-system/${MITM_POD}:root/.mitmproxy/mitmproxy-ca-cert.pem ${REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem
}

function main() {
  deploy_egress_proxy
  create_mitmproxy-ca-cert_pem_file
  green "HTTP Proxy CAcert:"
  cat ${REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem
}

main "$@"