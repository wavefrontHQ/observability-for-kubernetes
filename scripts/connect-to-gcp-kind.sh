#!/usr/bin/env bash

REPO_ROOT="$(git rev-parse --show-toplevel)"

function nukeRemoteKind() {
  scp "${REPO_ROOT}/make/kind-ha.yml" "root@${KIND_VM_IP}":/root/kind-config.yml

  ssh "root@${KIND_VM_IP}" \
    -- kind delete cluster
  ssh "root@${KIND_VM_IP}" \
    -- kind create cluster --config /root/kind-config.yml --image kindest/node:v1.24.7
}

function fetchKindPort() {
  ssh "root@${KIND_VM_IP}" \
    -- "docker ps | grep kind-external-load-balancer | grep -oP '127.0.0.1:\K\d*' | tr -d '\n'"
}

function main() {
  if [[ "$#" -ne 0 ]]; then
    while getopts "i:h:" opt; do
        case $opt in
        i)
          KIND_VM_SSH_PRIVATE_KEY=$OPTARG
          ;;
        h)
          KIND_VM_IP=$OPTARG
          ;;
        \?)
          print_usage_and_exit "Invalid option: -$OPTARG"
          ;;
        esac
      done
  fi

  if [[ -z "${KIND_VM_SSH_PRIVATE_KEY}" ]] || [[ -z "${KIND_VM_IP}" ]]; then
    echo "KIND_VM_SSH_PRIVATE_KEY and KIND_VM_IP need to be set or passed in as arguments"
    exit 1
  fi

  local known_hosts_filepath="${HOME}/.ssh/known_hosts"
  if [[ ! -f "${known_hosts_filepath}" ]]; then
    mkdir "${HOME}/.ssh"
    touch "${known_hosts_filepath}"
  fi

  if [[ -z "${SSH_AUTH_SOCK}" ]]; then
     eval "$(ssh-agent -s)"
  fi

  ssh-add "${KIND_VM_SSH_PRIVATE_KEY}"

  ssh-keygen -F "${KIND_VM_IP}" -f "${known_hosts_filepath}" | grep -q found || ssh-keyscan "${KIND_VM_IP}" >> "${known_hosts_filepath}" 2>/dev/null

  local kind_port=$(fetchKindPort)

  if [[ -z "${kind_port}" ]]; then
    nukeRemoteKind
    kind_port=$(fetchKindPort)
  fi
  echo "kind port: '${kind_port}'"

  mkdir -p /tmp
  scp "root@${KIND_VM_IP}":/root/.kube/config /tmp/.kubeconfig
  sed 's/kind-kind/gcp-kind/' /tmp/.kubeconfig > /tmp/.kubeconfig.bak
  mv /tmp/.kubeconfig{.bak,}

  ssh -NL "${kind_port}:localhost:${kind_port}" "root@${KIND_VM_IP}" &
  local tunnel_pid=$!
  echo "tunnel pid: '${tunnel_pid}'"
  echo "${tunnel_pid}" > /tmp/kind-tunnel-pid

  export KUBECONFIG="/tmp/.kubeconfig:${HOME}/.kube/config"

  kubectl config view --flatten > /tmp/combined-kubeconfig.yaml
  mkdir -p "${HOME}/.kube"
  mv /tmp/combined-kubeconfig.yaml "${HOME}/.kube/config"

  export KUBECONFIG="${HOME}/.kube/config"

  attempt_counter=0
  max_attempts=30

  echo "Waiting for gcp kind api server to be available through tunnel"
  while [[ "${attempt_counter}" -le "${max_attempts}" ]]; do
      curl curl --output /dev/null --silent --head --fail "https://127.0.0.1:${kind_port}" &> /dev/null
      if [[ "$?" -eq 0 ]]; then
          exit 0
      fi

      printf '.'
      attempt_counter=$(($attempt_counter+1))
      sleep 2
  done

  echo ""

  if [ ${attempt_counter} -eq ${max_attempts} ];then
    echo "Max attempts reached"
    exit 1
  fi

  kubectl config use-context gcp-kind
  kubectl get nodes
}

main $@
