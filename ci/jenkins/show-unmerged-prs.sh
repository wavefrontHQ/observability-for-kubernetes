#!/usr/bin/env bash

echo "Unmerged JIRA story PRs:"
echo "-------------------------------"

curl -sSL \
  -X GET \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  https://api.github.com/repos/wavefrontHQ/observability-for-kubernetes/pulls?state=open \
  | jq -r '.[] | select(.head.ref | startswith("K8SSAAS-")) | {ref: .head.ref, url: .html_url}' \
  | jq -r '"\(.url) | \(.ref)"'
