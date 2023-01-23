#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "${REPO_ROOT}"

function main() {

  cp operator/deploy/internal/collector/3-wavefront-collector-deployment.yaml.bak operator/deploy/internal/collector/3-wavefront-collector-deployment.yaml
  cp operator/deploy/internal/collector/2-wavefront-collector-daemonset.yaml.bak operator/deploy/internal/collector/2-wavefront-collector-daemonset.yaml
	rm operator/deploy/internal/collector/3-wavefront-collector-deployment.yaml.bak operator/deploy/internal/collector/2-wavefront-collector-daemonset.yaml.bak || true
}

main "$@"