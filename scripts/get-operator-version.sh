#!/usr/bin/env bash

REPO_ROOT=$(git rev-parse --show-toplevel)
cat "${REPO_ROOT}"/operator/release/OPERATOR_VERSION
