#!/usr/bin/env bash
set -eou pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
source "${REPO_ROOT}/scripts/k8s-utils.sh"
cd "$(dirname "$0")"

VERSION=$(cat ${REPO_ROOT}/operator/release/OPERATOR_VERSION)
GIT_BUMP_BRANCH_NAME="release-operator-${VERSION}"
git branch -D "$GIT_BUMP_BRANCH_NAME" &>/dev/null || true
git checkout -b "$GIT_BUMP_BRANCH_NAME"

git commit -am "Release operator version: ${VERSION}"
git push --force --set-upstream origin "${GIT_BUMP_BRANCH_NAME}"

#TODO Uncomment the below before merge
#PR_URL=$(curl \
#  -X POST \
#  -H "Authorization: token ${GITHUB_TOKEN}" \
#  -d "{\"head\":\"${GIT_BUMP_BRANCH_NAME}\",\"base\":\"main\",\"title\":\"Bump operator version to ${VERSION}\"}" \
#  https://api.github.com/repos/wavefrontHQ/observability-for-kubernetes/pulls |
#  jq -r '.url')

#echo "PR URL: ${PR_URL}"

#curl \
#  -X PUT \
#  -H "Authorization: token ${GITHUB_TOKEN}" \
#  -H "Accept: application/vnd.github.v3+json" \
#  "${PR_URL}/merge" \
#  -d "{\"commit_title\":\"Bump operator version to ${VERSION}\", \"merge_method\":\"squash\"}"