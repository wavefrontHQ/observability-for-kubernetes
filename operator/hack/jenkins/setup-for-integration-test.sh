#!/bin/bash
set -eou pipefail
#
# gcloud
#
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
# crane
#
if ! [ -x "$(command -v crane)" ]; then
  curl -H "Authorization: token ${GITHUB_TOKEN}" -L -s "https://github.com/google/go-containerregistry/releases/download/v0.13.0/go-containerregistry_Linux_x86_64.tar.gz" \
  | tar --to-stdout -xz crane \
  | sudo tee /usr/local/bin/crane >/dev/null
  sudo chmod +x /usr/local/bin/crane
fi
