#!/bin/bash -e


function print_usage_and_exit() {
    echo "Failure: $1"
    echo "Usage: $0 [flags] [options]"
    echo -e "\t-k kubernetes environment: gke or eks (required)"
    exit 1
}

while getopts ":k:" opt; do
  case $opt in
  k)
    K8S_ENV="$OPTARG"
    ;;
  \?)
    print_usage_and_exit "Invalid option: -$OPTARG"
    ;;
  esac
done

if [[ -z ${K8S_ENV} ]]; then
  print_usage_and_exit "kubernetes environment selection required"
fi

if [[ "${K8S_ENV}" == "gke" ]]; then
  gcloud auth activate-service-account --key-file "$GCP_CREDS"
  gcloud config set project wavefront-gcp-dev

  curl -fsSL "https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v2.0.0/docker-credential-gcr_linux_amd64-2.0.0.tar.gz" \
    | tar xz --to-stdout ./docker-credential-gcr | sudo tee /usr/local/bin/docker-credential-gcr >/dev/null
  sudo chmod +x /usr/local/bin/docker-credential-gcr
  docker-credential-gcr config --token-source="gcloud"
  docker-credential-gcr configure-docker --registries="us.gcr.io"
  (echo "https://us.gcr.io" | docker-credential-gcr get >/dev/null) \
    || (echo "docker credentials not configured properly"; exit 1)
fi

if [[ "${K8S_ENV}" == "eks" ]]; then
  if ! [ -x "$(command -v aws)" ]; then
    curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
    unzip awscliv2.zip
    sudo ./aws/install >/dev/null;
  fi
fi

if [[ "${K8S_ENV}" == "kind" ]]; then
  if ! [ -x "$(command -v kind)" ]; then
    # For AMD64 / x86_64
    [ $(uname -m) = x86_64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
    # For ARM64
    [ $(uname -m) = aarch64 ] && curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-arm64
    chmod +x ./kind
    sudo mv ./kind /usr/local/bin/kind
  fi
fi

#
# jq
#
if ! [ -x "$(command -v jq)" ]; then
  curl -H "Authorization: token ${GITHUB_TOKEN}" -L "https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64" > ./jq
  chmod +x ./jq
  sudo mv ./jq /usr/local/bin
fi
