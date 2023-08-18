#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)

function main() {
  cd "${REPO_ROOT}"

  gcloud config set project wavefront-gcp-dev

  CLUSTERS_TO_REMOVE=$(gcloud container clusters list --filter="resourceLabels.delete-me=true" --format json | jq -c '.[] | {name,zone}')

  local name zone
  for cluster in ${CLUSTERS_TO_REMOVE[@]}; do
    name=$(echo ${cluster} | jq -r '.name')
    zone=$(echo ${cluster} | jq -r '.zone')
    zone=${zone/us-central1-/}
    GKE_CLUSTER_NAME=${name} GCP_ZONE=${zone} make delete-gke-cluster || true
  done
}

main "$@"
