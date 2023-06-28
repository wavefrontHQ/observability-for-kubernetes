#!/usr/bin/env bash
set -euo pipefail

NS=$(kubectl get namespaces | awk '/collector-targets/ {print $1}')
if [ -z ${NS} ]; then exit 0; fi

SCRIPT_DIR="$(dirname "$0")"

cd "$SCRIPT_DIR"

echo "Uninstalling targets..."

kubectl patch -n collector-targets pod/pod-stuck-in-terminating --type=json -p '[{"op": "remove", "path": "/metadata/finalizers" }]' &>/dev/null || true

kubectl delete namespace collector-targets &>/dev/null || true

echo "Targets uninstalled"