#!/bin/bash
set -eou pipefail
#
# gcloud
#
if ! [ -x "$(command -v gcloud)" ]; then
  curl https://sdk.cloud.google.com > install.sh
  chmod +x ./install.sh
  sudo PREFIX=$HOME ./install.sh --disable-prompts >/dev/null;
fi

sudo /home/worker/google-cloud-sdk/bin/gcloud components install gke-gcloud-auth-plugin >/dev/null || gcloud components install gke-gcloud-auth-plugin || true
gcloud auth activate-service-account --key-file "$GCP_CREDS"
gcloud config set project wavefront-gcp-dev

#
# aws
#
if ! [ -x "$(command -v aws)" ]; then
  curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
  unzip awscliv2.zip
  sudo ./aws/install >/dev/null;
fi

#
# docker-credential-gcr
#
curl -fsSL "https://github.com/GoogleCloudPlatform/docker-credential-gcr/releases/download/v2.0.0/docker-credential-gcr_linux_amd64-2.0.0.tar.gz" \
  | tar xz --to-stdout ./docker-credential-gcr | sudo tee /usr/local/bin/docker-credential-gcr >/dev/null
sudo chmod +x /usr/local/bin/docker-credential-gcr
docker-credential-gcr config --token-source="gcloud"
docker-credential-gcr configure-docker --registries="us.gcr.io"
(echo "https://us.gcr.io" | docker-credential-gcr get >/dev/null) \
  || (echo "docker credentials not configured properly"; exit 1)

#
# kubectl
#
#curl -LO "https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl"
curl -LO "https://storage.googleapis.com/kubernetes-release/release/v1.23.6/bin/linux/amd64/kubectl"
chmod +x ./kubectl
sudo mv ./kubectl /usr/local/bin/kubectl

#
# jq
#
if ! [ -x "$(command -v jq)" ]; then
  curl -H "Authorization: token ${GITHUB_TOKEN}" -L "https://github.com/stedolan/jq/releases/download/jq-1.6/jq-linux64" > ./jq
  chmod +x ./jq
  sudo mv ./jq /usr/local/bin
fi

#
# yq
#
if ! [ -x "$(command -v yq)" ]; then
  curl -H "Authorization: token ${GITHUB_TOKEN}" -L "https://github.com/mikefarah/yq/releases/download/v4.26.1/yq_$(go env GOOS)_$(go env GOARCH)" > ./yq
  chmod +x ./yq
  sudo mv ./yq /usr/local/bin
fi

#
# kustomize
#
if ! [ -x "$(command -v kustomize)" ]; then
  curl -H "Authorization: token ${GITHUB_TOKEN}" -L -s "https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv4.4.0/kustomize_v4.4.0_linux_amd64.tar.gz" \
    | tar xz --to-stdout \
    | sudo tee /usr/local/bin/kustomize >/dev/null
  sudo chmod +x /usr/local/bin/kustomize
fi

#
# crane
#
if ! [ -x "$(command -v crane)" ]; then
  curl -H "Authorization: token ${GITHUB_TOKEN}" -L -s "https://github.com/google/go-containerregistry/releases/download/v0.13.0/go-containerregistry_Linux_x86_64.tar.gz" \
  | tar --to-stdout -xz crane \
  | sudo tee /usr/local/bin/crane >/dev/null
  sudo chmod +x /usr/local/bin/crane
fi

#
# semver cli
#
git config --global http.sslVerify false
make semver-cli
git config --global http.sslVerify true