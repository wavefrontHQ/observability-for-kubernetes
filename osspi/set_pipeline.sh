#!/usr/bin/env bash
set -efuo pipefail

which fly || (
  echo "This requires fly to be installed"
  echo "Download the binary from https://github.com/concourse/concourse/releases or from the Runway Concourse: https://runway-ci-sfo.eng.vmware.com"
  exit 1
)

fly -t runway-ci-sfo sync || (
  echo "This requires the runway target to be set"
  echo "Create this target by running 'fly -t runway-ci-sfo login -c https://runway-ci-sfo.eng.vmware.com -n k8po'"
  exit 1
)

pipeline_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
OSM_ENVIRONMENT=${OSM_ENVIRONMENT:-"production"}
echo "using OSM_ENVIRONMENT: ${OSM_ENVIRONMENT}. Valid environments are beta and production"

fly --target runway-ci-sfo set-pipeline \
    --pipeline observability-for-kubernetes-osspi \
    --config "${pipeline_dir}/pipeline.yaml" \ \
    --var osm-environment="${OSM_ENVIRONMENT}"
