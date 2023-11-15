#!/usr/bin/env bash
set -eup pipefail

yq '.metadata.namespace = "observability-system"' <&0 |
  yq '.metadata.labels["app.kubernetes.io/name"] = "wavefront"' |
  yq '.metadata.labels["app.kubernetes.io/component"] = "pixie"'
