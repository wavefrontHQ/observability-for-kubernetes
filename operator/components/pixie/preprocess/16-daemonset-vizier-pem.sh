#!/usr/bin/env bash
set -eup pipefail

PROCESS_DIR=$(dirname $0)

"$PROCESS_DIR/update-images.sh" <&0 \
  | yq '(.spec.template.spec.containers[] | select(.name == "pem") | .resources) = {}' \
  | yq '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_TABLE_STORE_DATA_LIMIT_MB", "value": "{{ .TableStoreLimits.TotalMiB }}"}' \
  | yq '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_TABLE_STORE_HTTP_EVENTS_PERCENT", "value": "{{ .TableStoreLimits.HttpEventsPercent }}"}' \
  | yq '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_TABLE_STORE_STIRLING_ERROR_LIMIT_BYTES", "value": "0"}' \
  | yq '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_TABLE_STORE_PROC_EXIT_EVENTS_LIMIT_BYTES", "value": "0"}' \
  | yq '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PX_STIRLING_HTTP_BODY_LIMIT_BYTES", "value": "{{ .MaxHTTPBodyBytes }}"}' \
  | yq '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_STIRLING_MAX_BODY_BYTES", "value": "{{ .MaxHTTPBodyBytes }}"}' \
  | yq '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_STIRLING_SOURCES", "value": "{{ .StirlingSourcesEnv }}"}' \
  | yq '.metadata.annotations["wavefront.com/conditionally-provision"] = "{{ .TLSCertsSecretExists }}"'