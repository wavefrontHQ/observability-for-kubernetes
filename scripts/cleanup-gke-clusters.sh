#!/bin/bash
#set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)

function main() {
  gcloud auth activate-service-account --key-file "$GCP_CREDS"
  gcloud config set project wavefront-gcp-dev

  CLUSTERS_TO_REMOVE=$(gcloud container clusters list --filter="-resourceLabels.keep-me=true" | awk '(NR>1) {print $1}')

  cd ${REPO_ROOT}
  for cluster_name in "${CLUSTERS_TO_REMOVE[@]}"
  do
    GKE_CLUSTER_NAME=${cluster_name} make delete-gke-cluster || true
  done
}

main $@
