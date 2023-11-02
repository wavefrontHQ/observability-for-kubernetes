#!/usr/bin/env bash
set -euo pipefail

# Default the environment variables if they are not already set
: "${PS_TRAFFIC_SCALE_FACTOR:=1.5}"
: "${PS_SAMPLE_PERIOD_MINUTES:=60}"
: "${PIXIE_SIZER_YAML:=TODO_URL_OF_LATEST_RELEASED_PIXIE_SIZER_YAML}"

# Deploy Pixie Sizer
sed "s/PS_TRAFFIC_SCALE_FACTOR_VALUE/$PS_TRAFFIC_SCALE_FACTOR/g" "$PIXIE_SIZER_YAML" | \
  sed "s/PS_SAMPLE_PERIOD_MINUTES_VALUE/$PS_SAMPLE_PERIOD_MINUTES/g" | \
  kubectl apply -f -

# Give useful commands
echo
echo "Wait $PS_SAMPLE_PERIOD_MINUTES minutes, then run the following command to see the recommendation:"
echo "    kubectl --namespace observability-system logs --selector=\"app.kubernetes.io/component=pixie-sizer\" --container=pixie-sizer --since=\"${PS_SAMPLE_PERIOD_MINUTES}m\""

echo
echo "Run the following command to completely remove the Pixie sizer installation:"
echo "    kubectl --namespace observability-system delete --all --selector=\"app.kubernetes.io/component=pixie-sizer\""

