#!/usr/bin/env bash
set -ex

cd "$(dirname "$0")" # cd to directory that create-and-merge-pull-request.sh is in

VERSION=$(cat ../../release/VERSION)
GIT_BUMP_BRANCH_NAME="bump-collector-${VERSION}"

PR_URL=$(curl \
  -X POST \
  -H "Authorization: token ${TOKEN}" \
  -d "{\"head\":\"${GIT_BUMP_BRANCH_NAME}\",\"base\":\"main\",\"title\":\"Bump collector version to ${VERSION}\"}" \
  https://api.github.com/repos/wavefrontHQ/observability-for-kubernetes/pulls |
  jq -r '.url')

echo "PR URL: ${PR_URL}"

curl \
  -X PUT \
  -H "Authorization: token ${TOKEN}" \
  -H "Accept: application/vnd.github.v3+json" \
  "${PR_URL}/merge" \
  -d "{\"commit_title\":\"Bump collector version to ${VERSION}\"}"
