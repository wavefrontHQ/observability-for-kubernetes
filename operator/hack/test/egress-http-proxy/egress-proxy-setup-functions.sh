#!/bin/bash -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

OPERATOR_REPO_ROOT=$(git rev-parse --show-toplevel)/operator

NS=observability-system

function deploy_egress_proxy() {
  kubectl create namespace ${NS} > /dev/null 2>&1 || true

  # deploy the mitmproxy
  kubectl apply -f ${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/egress-proxy.yaml

  # wait for egress proxy
  wait_for_cluster_ready "$NS"

  sleep 1
}

function delete_egress_proxy() {
  kubectl delete -f ${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/egress-proxy.yaml > /dev/null 2>&1 || true
  kubectl delete -f ${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/https-proxy-secret.yaml > /dev/null 2>&1 || true
  rm ${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem > /dev/null 2>&1 || true
}

function create_mitmproxy-ca-cert_pem_file() {
  #get httpproxy ip
  export MITM_IP="$(kubectl -n ${NS} get services --selector=app=egress-proxy -o jsonpath='{.items[*].spec.clusterIP}')"
  green "HTTP Proxy URL:"
  echo "http://${MITM_IP}:8080"

  # get the ca cert for the mitmproxy
  export MITM_POD="$(kubectl -n ${NS} get pods --selector=app=egress-proxy -o jsonpath='{.items[*].metadata.name}')"
  kubectl cp ${NS}/${MITM_POD}:root/.mitmproxy/mitmproxy-ca-cert.pem ${OPERATOR_REPO_ROOT}/hack/test/egress-http-proxy/mitmproxy-ca-cert.pem
}
