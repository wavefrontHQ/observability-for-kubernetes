#!/usr/bin/env bash
set -eou pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
source "${REPO_ROOT}/scripts/k8s-utils.sh"

operator_yaml="${REPO_ROOT}/deploy/wavefront-operator.yaml"

VERSION=$(cat ${REPO_ROOT}/operator/release/OPERATOR_VERSION)
GITHUB_REPO=wavefrontHQ/observability-for-kubernetes
AUTH="Authorization: token ${GITHUB_TOKEN}"

id=$(curl --fail -X POST -H "Content-Type:application/json" \
-H "$AUTH" \
-d "{
      \"tag_name\": \"v$VERSION\",
      \"target_commitish\": \"$GIT_BRANCH\",
      \"name\": \"Release v$VERSION\",
      \"body\": \"Description for v$VERSION\",
      \"draft\": true,
      \"prerelease\": false}" \
"https://api.github.com/repos/$GITHUB_REPO/releases" | jq ".id")

curl --data-binary @"$operator_yaml" \
  -H "$AUTH" \
  -H "Content-Type: application/octet-stream" \
"https://uploads.github.com/repos/$GITHUB_REPO/releases/$id/assets?name=$(basename $operator_yaml)"
