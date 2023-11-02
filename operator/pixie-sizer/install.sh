#!/usr/bin/env bash
set -euo pipefail

# Default the environment variables if they are not already set
: "${PS_TRAFFIC_SCALE_FACTOR:=1.5}"
: "${PS_SAMPLE_PERIOD_MINUTES:=60}"
: "${PIXIE_SIZER_YAML:=https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/operator/pixie-sizer/pixie-sizer-0.1.0.yaml}"

# Deploy Pixie Sizer
curl --silent -o - "$PIXIE_SIZER_YAML" | \
  sed "s/PS_TRAFFIC_SCALE_FACTOR_VALUE/$PS_TRAFFIC_SCALE_FACTOR/g" | \
  sed "s/PS_SAMPLE_PERIOD_MINUTES_VALUE/$PS_SAMPLE_PERIOD_MINUTES/g" | \
  kubectl apply -f -

# Give useful commands
echo
echo "Wait $PS_SAMPLE_PERIOD_MINUTES minutes, then run the following command to see the recommendation:"
echo "    kubectl --namespace observability-system logs --selector=\"app.kubernetes.io/component=pixie-sizer\" --container=pixie-sizer --since=\"${PS_SAMPLE_PERIOD_MINUTES}m\""

echo
echo "Run the following command to completely remove the Pixie sizer installation:"
echo "    kubectl --namespace observability-system delete --all --selector=\"app.kubernetes.io/component=pixie-sizer\""

