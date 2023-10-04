#!/usr/bin/env bash
set -eup pipefail

sed 's/  PL_CLUSTER_NAME: ""/  PL_CLUSTER_NAME: {{ .ClusterName }}/' <&0