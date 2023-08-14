#!/bin/bash
set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)

function main() {
#  TODO: to make this right
  CLUSTERS_TO_REMOVE=$(gcloud compute instances list  --filter="labels.test" --project=wavefront-gcp-dev)

  cd ${REPO_ROOT}

  for cluster_name in "${CLUSTERS_TO_REMOVE[@]}" ; do
    GKE_CLUSTER_NAME=${cluster_name} make delete-gke-cluster
  done
}

main $@
