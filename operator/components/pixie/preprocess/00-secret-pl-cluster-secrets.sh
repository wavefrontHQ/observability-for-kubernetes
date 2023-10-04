#!/usr/bin/env bash
set -eup pipefail

yq '.stringData.cluster-id = "{{ .ClusterUUID }}"' <&0 \
  | yq '.stringData.cluster-name = "{{ .ClusterName }}"'