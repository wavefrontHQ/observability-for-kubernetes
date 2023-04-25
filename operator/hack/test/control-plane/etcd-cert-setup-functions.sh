#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

OPERATOR_DIR="${REPO_ROOT}/operator"

NS=observability-system

function deploy_etcd_cert_printer() {
  kubectl create namespace ${NS} > /dev/null 2>&1 || true

  # deploy the etcd cert printer
  kubectl delete -f ${OPERATOR_DIR}/hack/test/control-plane/etcd-cert-printer.yaml -n "${NS}" || true
  kubectl apply -f ${OPERATOR_DIR}/hack/test/control-plane/etcd-cert-printer.yaml -n "${NS}"

  # wait for etcd cert printer
  wait_for_cluster_ready "$NS"

  sleep 1
}

function delete_etcd_cert_printer() {
  kubectl delete -f ${OPERATOR_DIR}/hack/test/control-plane/etcd-cert-printer.yaml > /dev/null 2>&1 || true
  kubectl delete -f ${OPERATOR_DIR}/hack/test/control-plane/etcd-certs-secret.yaml > /dev/null 2>&1 || true
  rm "${OPERATOR_DIR}/build/ca.crt" > /dev/null 2>&1 || true
  rm "${OPERATOR_DIR}/build/server.crt" > /dev/null 2>&1 || true
  rm "${OPERATOR_DIR}/build/server.key" > /dev/null 2>&1 || true
}

function create_etcd_cert_files() {
  # get the control plane pod name
  POD_NAME=$(kubectl get pods -n "${NS}" | awk 'NR==2, /etcd-cert-printer/ {print $1}')
  echo "POD NAME: $POD_NAME"

  kubectl wait --for=condition=Ready pod/$POD_NAME -n "${NS}" --timeout=5s &>/dev/null

  # get the control plane etcd certs
  kubectl logs ${POD_NAME} -n "${NS}" > "${OPERATOR_DIR}/build/all_certs.txt"
  kubectl delete -f ${OPERATOR_DIR}/hack/test/control-plane/etcd-cert-printer.yaml &>/dev/null || true

  csplit "${OPERATOR_DIR}/build/all_certs.txt" \
    '/^-----BEGIN CERTIFICATE-----$/' '/^-----BEGIN RSA PRIVATE KEY-----$/' &>/dev/null
  mv xx00 "${OPERATOR_DIR}/build/ca.crt"
  mv xx01 "${OPERATOR_DIR}/build/server.crt"
  mv xx02 "${OPERATOR_DIR}/build/server.key"
  rm "${OPERATOR_DIR}/build/all_certs.txt" || true
}
