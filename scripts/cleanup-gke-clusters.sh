#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)

function main() {
  cd "${REPO_ROOT}"
  gcloud auth activate-service-account --key-file "$GCP_CREDS"
  gcloud config set project wavefront-gcp-dev
  CLUSTERS_TO_REMOVE=$(\
    gcloud container clusters list \
    --project wavefront-gcp-dev \
    --filter="resourceLabels.delete-me=true" \
    --format="csv[no-heading](name,zone,resourceLabels)"\
  )

  local name zone expires_in_days expired_time
  for cluster in ${CLUSTERS_TO_REMOVE[@]}; do
    name=$(echo ${cluster} | cut -d ',' -f 1)
    zone=$(echo ${cluster} | cut -d ',' -f 2 | cut -d '-' -f 3)

    if echo "${cluster}" | grep 'expires-in-days' >/dev/null; then
      expires_in_days=$(\
        echo ${cluster} \
        | cut -d ',' -f 3 \
        | sed 's/.*expires-in-days=//g'\
      )

      expired_time=$(date --date="${expires_in_days} day ago" +%s)
      cluster=$(gcloud container clusters list \
        --project wavefront-gcp-dev \
        --filter="createTime.date(\"+%s\")<${expired_time}" \
        --format="csv[no-heading](name,zone)" \
        | grep "${name}" || echo "")
      if [ -z "${cluster}" ]; then
        continue
      fi
    fi

    GKE_CLUSTER_NAME=${name} GCP_ZONE=${zone} GKE_WAIT_FOR_COMPLETE=false make delete-gke-cluster
  done
}

main "$@"
