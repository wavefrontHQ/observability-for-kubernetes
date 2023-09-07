#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)

function main() {
  cd "${REPO_ROOT}"
  gcloud auth activate-service-account --key-file "$GCP_CREDS"
  gcloud config set project wavefront-gcp-dev
  CLUSTERS_TO_REMOVE=$(gcloud container clusters list \
    --project wavefront-gcp-dev \
    --filter="resourceLabels.delete-me=true" \
    --format="csv[no-heading](name,zone,resourceLabels)")

  echo "CLUSTERS_TO_REMOVE: ${CLUSTERS_TO_REMOVE}"

  local name zone expires_in_days expired_creation_time
  for cluster in ${CLUSTERS_TO_REMOVE[@]}; do
    name=$(echo ${cluster} | cut -d ',' -f 1)
    zone=$(echo ${cluster} | cut -d ',' -f 2 | cut -d '-' -f 3)
    expires_in_days=$(echo ${cluster} | cut -d ',' -f 3 | sed 's/.*expires-in-days=//g')
    echo "extracting name '${name}', zone '${zone}', and expires-in-days '${expires_in_days}' from '${cluster}'"

    expired_creation_time=$(date --date="${expires_in_days} day ago" +%y%m%d)
    echo "now vs expired_creation_time: $(date +%s) vs ${expired_creation_time}"
    gcloud container clusters list \
        --project wavefront-gcp-dev \
        --filter="name=${name},creationTimestamp.date(\"+%s\")<${expired_creation_time}" \
        --format="csv[no-heading](name,zone)"
#    GKE_CLUSTER_NAME=${name} GCP_ZONE=${zone} GKE_WAIT_FOR_COMPLETE=false make delete-gke-cluster &
  done
}

main "$@"
