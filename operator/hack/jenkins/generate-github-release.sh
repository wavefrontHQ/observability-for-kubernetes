#!/usr/bin/env bash
set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

function main() {
  local VERSION
  VERSION=$(get_operator_version)
  local collector proxy logging
  collector=$(get_component_version 'collector')
  proxy=$(get_component_version 'proxy')
  logging=$(get_component_version 'logging')

  local GITHUB_REPO=wavefrontHQ/observability-for-kubernetes
  local AUTH="Authorization: Bearer ${GITHUB_TOKEN}"
  local BODY="Description for v${VERSION}\n\n|Component|Version|\n|---|---|\n|Wavefront Kubernetes Collector|${collector}|\n|Wavefront Proxy|${proxy}|\n|Fluent Bit|${logging}|\n"

  local response
  response=$(curl -fsSL \
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
    | jq '{id, html_url}')

  local release_url
  release_url=$(echo "${response}" | jq -r '.html_url')
  echo "Release URL: ${release_url}"

  local id
  id=$(echo "$response" | jq -r '.id')
  local operator_yaml="${REPO_ROOT}/deploy/wavefront-operator.yaml"

  local download_url
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
}

main "$@"
