#!/usr/bin/env bash
set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

operator_yaml="${REPO_ROOT}/deploy/wavefront-operator.yaml"
collector=$(get_component_version 'collector')
proxy=$(get_component_version 'proxy')
logging=$(get_component_version 'logging')

VERSION=$(cat "${REPO_ROOT}/operator/release/OPERATOR_VERSION")
GITHUB_REPO=wavefrontHQ/observability-for-kubernetes
AUTH="Authorization: Bearer ${GITHUB_TOKEN}"
BODY="Description for v${VERSION}\n\n|Component|Version|\n|---|---|\n|Wavefront Kubernetes Collector|${collector}|\n|Wavefront Proxy|${proxy}|\n|Fluent Bit|${logging}|\n"

id=$(curl -fsSL \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "$AUTH" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "https://api.github.com/repos/$GITHUB_REPO/releases" \
  -d "{
      \"tag_name\": \"v$VERSION\",
      \"target_commitish\": \"main\",
      \"name\": \"Release v$VERSION\",
      \"body\": \"${BODY}\",
      \"draft\": true,
      \"prerelease\": false,
      \"generate_release_notes\": true
      }" \
  | jq '.id')

download_url=$(curl -sSL \
  -X POST \
  --data-binary @"$operator_yaml" \
  -H "Accept: application/vnd.github+json" \
  -H "$AUTH" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Content-Type: application/octet-stream" \
  "https://uploads.github.com/repos/$GITHUB_REPO/releases/$id/assets?name=$(basename $operator_yaml)" \
  | jq -r '.browser_download_url')

echo "Download URL: ${download_url}"
