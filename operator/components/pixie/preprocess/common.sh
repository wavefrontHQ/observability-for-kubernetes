#!/usr/bin/env bash
set -eup pipefail

yq '.metadata.namespace = "observability-system"' <&0
