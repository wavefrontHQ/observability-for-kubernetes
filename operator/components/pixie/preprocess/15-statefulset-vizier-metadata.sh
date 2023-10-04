#!/usr/bin/env bash
set -eup pipefail

PROCESS_DIR=$(dirname $0)

"$PROCESS_DIR/update-images.sh" <&0 \
  | yq '(.spec.template.spec.containers[] | select(.name == "app") | .resources) = {}' \
  | yq '.metadata.annotations["wavefront.com/conditionally-provision"] = "{{ .TLSCertsSecretExists }}"'