#!/bin/bash
set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)

function main() {
  # uncomment this by the end of the week
#  CLUSTERS_TO_REMOVE=$(gcloud container clusters list --filter="-resourceLabels.keep-me=true")
  CLUSTERS_TO_REMOVE=$(gcloud container clusters list --filter="resourceLabels.test=true")
  cd ${REPO_ROOT}

  for cluster_name in "${CLUSTERS_TO_REMOVE[@]}" ; do
    GKE_CLUSTER_NAME=${cluster_name} make delete-gke-cluster
  done
}

main $@
