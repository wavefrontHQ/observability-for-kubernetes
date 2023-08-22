#!/usr/bin/env bash
set -euo pipefail

function print_usage_and_exit() {
  echo "Failure: $1"
  echo "Usage: $0 -k <K8S_ENV>"
  echo -e "\t-k kubernetes environment (optional, ex: 'gke' or 'eks')"
  exit 1
}

K8S_ENV=''

while getopts 'k:' opt; do
  case "${opt}" in
    k) K8S_ENV="${OPTARG}";;
    \?) print_usage_and_exit "Invalid option";;
  esac
done

if [[ "${K8S_ENV}" == "eks" ]]; then
  if ! [ -x "$(command -v aws)" ]; then
    curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
    unzip awscliv2.zip
    sudo ./aws/install >/dev/null;
  fi
fi

GCR_VERSION='2.0.0'

if [[ "${K8S_ENV}" == "gke" ]]; then
  gcloud auth activate-service-account --key-file "$GCP_CREDS"
  gcloud config set project wavefront-gcp-dev

  curl -fsSL "https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v${GCR_VERSION}/docker-credential-gcr_linux_amd64-${GCR_VERSION}.tar.gz" \
    | tar xz --to-stdout ./docker-credential-gcr \
    | sudo tee /usr/local/bin/docker-credential-gcr >/dev/null
  sudo chmod +x /usr/local/bin/docker-credential-gcr
  docker-credential-gcr config --token-source="gcloud"
  docker-credential-gcr configure-docker --registries="us.gcr.io"
  (echo "https://us.gcr.io" | docker-credential-gcr get >/dev/null) \
    || (echo "docker credentials not configured properly"; exit 1)
fi

if [[ "${K8S_ENV}" == "tkgm" ]]; then
  if [[ ! -d "$KUBECONFIG_DIR" ]]; then
    mkdir -p "$KUBECONFIG_DIR"
  fi
  if [[ ! -f "$KUBECONFIG_FILE" ]]; then
    echo "" > "$KUBECONFIG_FILE"
    chmod go-r "$KUBECONFIG_FILE"
  fi
  sheepctl -n k8po-team lock list -j \
    | jq -r '. | map(select(.status == \"locked\" and .pool_name != null and (.pool_name | contains(\"tkg\")))) | .[0].access' \
    | jq -r '.tkg[0].kubeconfig' \
    > "$KUBECONFIG_FILE"
fi

JQ_VERSION='1.6'

if ! [ -x "$(command -v jq)" ]; then
  echo "Installing jq ..."
  curl -H "Authorization: token ${GITHUB_TOKEN}" \
    -sSL "https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64" > ./jq
  chmod +x ./jq
  sudo mv ./jq /usr/local/bin
fi

YQ_VERSION='4.26.1'

if ! [ -x "$(command -v yq)" ]; then
  echo "Installing yq ..."
  curl -H "Authorization: token ${GITHUB_TOKEN}" \
    -sSL "https://github.com/mikefarah/yq/releases/download/v${YQ_VERSION}/yq_linux_amd64" > ./yq
  chmod +x ./yq
  sudo mv ./yq /usr/local/bin
fi
