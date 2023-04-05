#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

OPERATOR_REPO_ROOT="${REPO_ROOT}/operator"

NS=observability-system

function deploy_etcd_certs_printer() {
  kubectl create namespace ${NS} > /dev/null 2>&1 || true

  # deploy the etcd cert printer
  kubectl apply -f ${OPERATOR_REPO_ROOT}/hack/test/control-plane/etcd-cert-printer.yaml

  # wait for etcd cert
  wait_for_cluster_ready "$NS"

  sleep 1


}

function delete_() {
    kubectl delete -f ${OPERATOR_REPO_ROOT}/hack/test/control-plane/etcd-cert-printer.yaml
}

function create_etcd_cert_file() {
    POD_NAME=$(kubectl get pods -n default | awk '/etcd-cert-printer/ {print $1}')
    kubectl logs -p ${POD_NAME} > "all_certs.txt"
    csplit all_certs.txt '/^---$/' &>/dev/null
    mv xx00 ${OPERATOR_REPO_ROOT}/hack/test/control-plane/ca.crt
    mv xx01 ${OPERATOR_REPO_ROOT}/hack/test/control-plane/server.crt
    mv xx02 ${OPERATOR_REPO_ROOT}/hack/test/control-plane/server.key
}