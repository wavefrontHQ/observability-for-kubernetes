#!/usr/bin/env bash
set -euo pipefail

if [ -z "$STREAM_NAME" ]; then
  echo "Set STREAM_NAME before running this script"
  exit 1
fi
LEMANS_GATEWAY_SERVER="localhost:8002"
CSP_SECRET="$(echo -n "$(cat tmp/csp_username):$(cat tmp/csp_password)" | base64)"

token_file=$(mktemp)

echo "authenticating"
curl --fail-with-body --location --request POST 'https://console-stg.cloud.vmware.com/csp/gateway/am/api/auth/authorize' \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --header "Authorization: Basic $CSP_SECRET" \
  --data-urlencode 'grant_type=client_credentials' \
  -o "$token_file"

CSP_AUTH_TOKEN="$(gojq -r .access_token  "$token_file")"


curl --location --fail-with-body --request POST "http://$LEMANS_GATEWAY_SERVER/le-mans/v1/streams/$STREAM_NAME" \
  --header "x-xenon-auth-token: #CSP#$CSP_AUTH_TOKEN" \
  --header 'Content-Type: text/plain' \
  --data "test.luke-to-lemans-via-troll 100 source=luke-dev-mac"
