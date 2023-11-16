#!/usr/bin/env bash
set -eup pipefail

function update_subject_if_necessary() {
    local input="$(</dev/stdin)"
    if [[ "$(yq '. | has("subjects")' <<< "$input")" == "true" ]]; then
      yq '(.subjects[] | .namespace) = "observability-system"' <<< "$input"
    else
      echo "$input"
    fi
}

yq '.metadata.namespace = "observability-system"' <&0 |
  yq '.metadata.labels["app.kubernetes.io/name"] = "wavefront"' |
  yq '.metadata.labels["app.kubernetes.io/component"] = "pixie"' |
  update_subject_if_necessary
