#!/usr/bin/env bash
set -euo pipefail

LEMANS_RESOURCE_SERVER="localhost:8001"
CSP_SECRET="$(echo -n "$(cat tmp/csp_username):$(cat tmp/csp_password)" | base64)"

token_file=$(mktemp)
lemans_token_file=$(mktemp)

curl -s --fail-with-body --location --request POST 'https://console-stg.cloud.vmware.com/csp/gateway/am/api/auth/authorize' \
  --header 'Content-Type: application/x-www-form-urlencoded' \
  --header "Authorization: Basic $CSP_SECRET" \
  --data-urlencode 'grant_type=client_credentials' \
  -o "$token_file"

CSP_AUTH_TOKEN="$(gojq -r .access_token  "$token_file")"

curl -s --location --fail-with-body --request POST "http://$LEMANS_RESOURCE_SERVER/le-mans/v2/resources/access-keys" \
  --header "x-xenon-auth-token: $CSP_AUTH_TOKEN" \
  --header 'Content-Type: text/plain' \
  --data '{ "name": "test-access-key1", "orgId": "Org-ID","createdBy": "csp@vmware.com" }' \
  -o "$lemans_token_file"

LEMANS_API_TOKEN="$(gojq -r .key  "$lemans_token_file")"

echo $LEMANS_API_TOKEN