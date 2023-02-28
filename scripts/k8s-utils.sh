REPO_ROOT=$(git rev-parse --show-toplevel)

function green() {
  echo -e $'\e[32m'"${1}"$'\e[0m'
}

function red() {
  echo -e $'\e[31m'"${1}"$'\e[0m'
}

function yellow() {
  echo -e $'\e[1;33m'"${1}"$'\e[0m'
}

function print_msg_and_exit() {
  red "${1}"
  exit 1
}

function pushd_check() {
  local d="${1}"
  pushd ${d} || print_msg_and_exit "Entering directory '${d}' with 'pushd' failed!"
}

function popd_check() {
  local d="${1}"
  popd || print_msg_and_exit "Leaving '${d}' with 'popd' failed!"
}

function wait_for_cluster_ready() {
  if [ -z "${1}" ]; then
    printf "Waiting for all Pods to be 'Ready' ..."
    while ! kubectl wait --for=condition=Ready pod --all -l exclude-me!=true --all-namespaces --timeout=5s  &> /dev/null; do
      printf "."
      sleep 1
    done
  else
    local ns=${1}
    printf "Waiting for all Pods in \"${1}\" namespace to be 'Ready' ..."
    while ! kubectl wait --for=condition=Ready pod --all -l exclude-me!=true -n "$ns" --timeout=5s &> /dev/null; do
      printf "."
      sleep 1
    done
  fi

  echo " done."
}

function wait_for_namespace_created() {
	local namespace=$1
  printf "Waiting for namespace \"$1\" to be created ..."
  while ! kubectl create namespace collector-targets &> /dev/null; do
    printf "."
    sleep 1
  done
  echo " done."
}

function wait_for_namespaced_resource_created() {
  local namespace=$1
  local resource=$2
  printf "Waiting for $resource in $namespace to be created"
  while ! kubectl get --namespace $namespace $resource > /dev/null; do
    printf "."
    sleep 1
  done
  echo " done."
}

function wait_for_proxy_termination() {
  printf "Waiting for proxy to be terminated ..."
  local ns=${1:-observability-system}
  while ! kubectl wait --for=delete  -n "$ns" pod -l app.kubernetes.io/name=wavefront -l app.kubernetes.io/component=proxy --timeout=5s &> /dev/null; do
    printf "."
    sleep 1
  done
  echo " done."
}

function wait_for_cluster_resource_deleted() {
  local resource=$1
  printf "Waiting for $resource to be deleted"
  while kubectl get $resource &> /dev/null; do
    printf "."
    sleep 1
  done
  echo " done."
}

function create_cluster_name() {
  echo $(whoami)-$(k8s_env | awk '{print tolower($0)}')-operator-$(date +"%y%m%d")
}

function k8s_env() {
  "${REPO_ROOT}/scripts/get-k8s-cluster-env.sh"
}

function get_component_version() {
  local component_name=$1
  yq .data."${component_name}" "${REPO_ROOT}"/operator/config/manager/component_versions.yaml
}

function get_next_collector_version() {
  cat ${REPO_ROOT}/collector/release/NEXT_RELEASE_VERSION
}

function get_next_operator_version() {
  "${REPO_ROOT}"/scripts/get-next-operator-version.sh
}
