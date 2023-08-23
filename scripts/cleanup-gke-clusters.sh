#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)

function main() {
  cd "${REPO_ROOT}"

  CLUSTERS_TO_REMOVE=$(gcloud container clusters list --project wavefront-gcp-dev --filter="resourceLabels.delete-me=true" --format="csv[no-heading](name,zone)")

  local name zone
  for cluster in ${CLUSTERS_TO_REMOVE[@]}; do
    name=$(echo ${cluster} | cut -d ',' -f 1)
    zone=$(echo ${cluster} | cut -d ',' -f 2 | cut -d '-' -f 3)
    GKE_CLUSTER_NAME=${name} GCP_ZONE=${zone} make delete-gke-cluster || true
  done
}

main "$@"
