#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

OPERATOR_DIR="${REPO_ROOT}/operator"

NS=observability-system

function deploy_etcd_cert_printer() {
  kubectl create namespace ${NS} > /dev/null 2>&1 || true

  # deploy the etcd cert printer
  kubectl apply -f ${OPERATOR_DIR}/hack/test/control-plane/etcd-cert-printer.yaml

  # wait for etcd cert printer
  wait_for_cluster_ready "$NS"

  sleep 1
}

function delete_etcd_cert_printer() {
  kubectl delete -f ${OPERATOR_DIR}/hack/test/control-plane/etcd-cert-printer.yaml > /dev/null 2>&1 || true
  kubectl delete -f ${OPERATOR_DIR}/hack/test/control-plane/etcd-certs-secret.yaml > /dev/null 2>&1 || true
  rm ${OPERATOR_DIR}/hack/test/control-plane/ca.crt > /dev/null 2>&1 || true
  rm ${OPERATOR_DIR}/hack/test/control-plane/server.crt > /dev/null 2>&1 || true
  rm ${OPERATOR_DIR}/hack/test/control-plane/server.key > /dev/null 2>&1 || true
}

function create_etcd_cert_files() {
  # get the control plane pod name
  POD_NAME=$(kubectl get pods -n default | awk 'NR==2, /etcd-cert-printer/ {print $1}')
  echo "POD NAME: $POD_NAME"

  # get the control plane etcd certs
  kubectl logs -p ${POD_NAME} > "${OPERATOR_DIR}/hack/test/control-plane/all_certs.txt"
  csplit "${OPERATOR_DIR}/hack/test/control-plane/all_certs.txt" \
    '/^-----BEGIN CERTIFICATE-----$/' '/^-----BEGIN RSA PRIVATE KEY-----$/' &>/dev/null
  mv xx00 ${OPERATOR_DIR}/hack/test/control-plane/ca.crt
  mv xx01 ${OPERATOR_DIR}/hack/test/control-plane/server.crt
  mv xx02 ${OPERATOR_DIR}/hack/test/control-plane/server.key
  rm "${OPERATOR_DIR}/hack/test/control-plane/all_certs.txt"
}
