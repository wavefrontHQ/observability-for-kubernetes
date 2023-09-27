#!/usr/bin/env bash
set -euo pipefail

if [ -z "$STREAM_NAME" ]; then
  echo "Set STREAM_NAME before running this script"
  exit 1
fi
RECEIVER_NAME="$STREAM_NAME"
LEMANS_RESOURCE_SERVER="localhost:8001"
RECEIVER_URI=http://troll-lb.acceptance.svc.cluster.local:8000/report
CSP_SECRET="$(echo -n "$(cat tmp/le_mans_csp_client_id):$(cat tmp/le_mans_csp_client_secret)" | base64)"

token_file=$(mktemp)

echo "authenticating"
curl --fail-with-body --location --request POST 'https://console-stg.cloud.vmware.com/csp/gateway/am/api/auth/authorize' \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --header "Authorization: Basic $CSP_SECRET" \
  --data-urlencode 'grant_type=client_credentials' \
  -o "$token_file"

CSP_AUTH_TOKEN="$(gojq -r .access_token  "$token_file")"

echo "creating receiver"

receiver_json_file="$(mktemp)"
echo "$receiver_json_file"
cat <<-JSON >"$receiver_json_file"
{
  "name": "$RECEIVER_NAME",
  "address": "$RECEIVER_URI",
  "useHttp2": false
}
JSON

curl --location --fail-with-body --request POST "http://$LEMANS_RESOURCE_SERVER/le-mans/v2/resources/receivers" \
  --header "x-xenon-auth-token: $CSP_AUTH_TOKEN" \
  --header 'Content-Type: application/json' \
  --data @"$receiver_json_file"

echo "creating stream"

stream_json_file="$(mktemp)"
echo "$stream_json_file"

cat <<-JSON >"$stream_json_file"
{
    "name": "$STREAM_NAME",
    "deliveryPolicy": "WAIT_ALL",
    "receiverLinks": ["/le-mans/v2/resources/receivers/$RECEIVER_NAME"]
}
JSON

curl --location --fail-with-body --request POST "http://$LEMANS_RESOURCE_SERVER/le-mans/v2/resources/streams" \
  --header "x-xenon-auth-token: $CSP_AUTH_TOKEN" \
  --header 'Content-Type: application/json' \
  --data @"$stream_json_file"
