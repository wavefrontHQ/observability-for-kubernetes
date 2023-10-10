#!/usr/bin/env bash

OPEN_STORY_PRS=$(curl -sSL \
    -X GET \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer ${GITHUB_TOKEN}" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    https://api.github.com/repos/wavefrontHQ/observability-for-kubernetes/pulls?state=open \
    | jq -r .[].head.ref \
    | grep K8SSAAS-)

echo "Unmerged PR branches:"
for open_story_pr in ${OPEN_STORY_PRS} ; do
  echo "- ${open_story_pr}: https://github.com/wavefrontHQ/observability-for-kubernetes/tree/${open_story_pr}"
done
