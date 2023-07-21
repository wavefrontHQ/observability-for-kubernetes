#!/bin/bash -e

CURRENT_CONTEXT=$(kubectl config current-context 2>/dev/null)

if ! grep -q "kind" <<< "$CURRENT_CONTEXT"; then
  echo "-error (Not a kind environment)"
  exit 0
fi

if [[ "$CURRENT_CONTEXT" == "kind-kind" ]]; then
  echo "-local"
else
  echo "-remote"
fi
