#!/usr/bin/env bash
set -eup pipefail

PIXIE_CURL_IMG=gcr.io/pixie-oss/pixie-dev-public/curl:multiarch-7.87.0
TANZU_CURL_IMG=projects.registry.vmware.com/tanzu_observability/bitnami/os-shell:11

yq 'del(.spec.template.spec.initContainers[] | select(.name == "cc-wait") )' <&0 \
  | yq "(.spec.template.spec.initContainers[] | .image) |= sub(\"$PIXIE_CURL_IMG.*\",\"$TANZU_CURL_IMG\")" \
  | yq '(.spec.template.spec.containers[] | .image) |= sub("gcr.io/([^@]*)(@sha256.*)?$","projects.registry.vmware.com/tanzu_observability/$1-multi")'