#!/usr/bin/env bash
set -eou pipefail

REPO_ROOT="$(git rev-parse --show-toplevel)"
source "${REPO_ROOT}/scripts/k8s-utils.sh"
cd "$REPO_ROOT"

VERSION=$(cat "${REPO_ROOT}/operator/release/OPERATOR_VERSION")
GIT_BUMP_BRANCH_NAME="release-operator-${VERSION}"
git branch -D "$GIT_BUMP_BRANCH_NAME" &>/dev/null || true
git checkout -b "$GIT_BUMP_BRANCH_NAME"

# Add changes from promoting dev-internal folder
sed -i.bak "s%newTag:.*$%newTag: ${VERSION}%" operator/config/manager/kustomization.yaml
rm -f operator/config/manager/kustomization.yaml.bak
git add deploy/ docs/ README.md operator/config/manager/kustomization.yaml
git commit -am "Release operator version: ${VERSION}"
git push --force --set-upstream origin "${GIT_BUMP_BRANCH_NAME}"

PR_URL=$(curl -sSL \
  -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d "{\"head\":\"${GIT_BUMP_BRANCH_NAME}\",\"base\":\"main\",\"title\":\"Release operator version: ${VERSION}\"}" \
  https://api.github.com/repos/wavefrontHQ/observability-for-kubernetes/pulls |
  jq -r '.url')

echo "PR URL: ${PR_URL}"

curl -sSL \
  -X PUT \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  "${PR_URL}/merge" \
  -d "{\"commit_title\":\"Release operator version: ${VERSION}\", \"merge_method\":\"squash\"}"
