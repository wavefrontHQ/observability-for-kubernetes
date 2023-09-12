#!/usr/bin/env bash
set -e

REPO_ROOT=$(git rev-parse --show-toplevel)

function main() {
  cd "${REPO_ROOT}"
  gcloud auth activate-service-account --key-file "$GCP_CREDS"
  gcloud config set project wavefront-gcp-dev
  CLUSTERS=$(\
    gcloud container clusters list \
    --project wavefront-gcp-dev \
    --format="csv[no-heading](name,zone,resourceLabels)"\
  )

  local name zone labels
  # shellcheck disable=SC2068
  for cluster in ${CLUSTERS[@]}; do
    name=$(echo "${cluster}" | cut -d ',' -f 1)
    zone=$(echo "${cluster}" | cut -d ',' -f 2 | cut -d '-' -f 3)
    labels=$(echo "${cluster}" | cut -d ',' -f 3)

    if [ -z "${labels}" ]; then
      continue
    fi

    if ! echo "${labels}" | grep 'expire-date' >/dev/null; then
      continue
    fi
    echo "Found cluster '${name}' with expiration labels '${labels}'"

    local expire_date expire_time
    expire_date=$(date +%m-%d-%y)
    expire_time=$(date +%H%M%z)

    for label_pair in ${labels//;/ } ; do
      key=$(echo "${label_pair}" | cut -d '=' -f 1)
      value=$(echo "${label_pair}" | cut -d '=' -f 2)
      if [ "${key}" == "expire-date" ]; then
        expire_date="${value}"
      fi
      if [ "${key}" == "expire-time" ]; then
        expire_time=$(echo "${value}" | tr '_' ':')
      fi
    done

    local expire_datetime expire_timestamp now_timestamp
    expire_datetime=$(date --date="${expire_date} ${expire_time}")
    expire_timestamp=$(date --date="${expire_date} ${expire_time}" +%s)
    now_timestamp=$(date +%s)

    if [ "$expire_timestamp" -ge "$now_timestamp" ]; then
      continue
    fi
    echo "Deleting expired cluster '${name}' with expiration date '${expire_datetime}'"

    GKE_CLUSTER_NAME=${name} GCP_ZONE=${zone} GKE_ASYNC=true make delete-gke-cluster
  done
}

main "$@"
