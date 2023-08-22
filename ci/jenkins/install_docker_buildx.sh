#!/usr/bin/env bash
set -euo pipefail

BUILDX_VERSION='0.9.1'
CRANE_VERSION='0.13.0'

if [[ ! -f "$HOME/.docker/cli-plugins/docker-buildx" ]]; then
  echo "Installing docker buildx ..."
  wget -q -O docker-buildx \
    "https://github.com/docker/buildx/releases/download/v${BUILDX_VERSION}/buildx-v${BUILDX_VERSION}.linux-amd64"
  chmod a+x docker-buildx
  if [[ ! -d "$HOME/.docker/cli-plugins" ]]; then mkdir -p ~/.docker/cli-plugins; fi
  mv docker-buildx ~/.docker/cli-plugins
  echo "Successfully installed docker buildx: $(docker buildx version)"
else
  echo "buildx already installed: $(docker buildx version)"
fi

if ! [ -x "$(command -v crane)" ]; then
  echo "Installing crane ..."
  curl -H "Authorization: token ${GITHUB_TOKEN}" \
    -sSL "https://github.com/google/go-containerregistry/releases/download/v${CRANE_VERSION}/go-containerregistry_Linux_x86_64.tar.gz" \
    | tar --to-stdout -xz crane \
    | sudo tee /usr/local/bin/crane >/dev/null
  sudo chmod +x /usr/local/bin/crane
fi
