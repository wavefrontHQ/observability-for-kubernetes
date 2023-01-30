#!/usr/bin/env bash
set -e

cd "$(dirname "$0")"

COLLECTOR_REPO_ROOT=$(git rev-parse --show-toplevel)/collector
TEMP_DIR=$(mktemp -d)
NS=wavefront-collector

cp "$COLLECTOR_REPO_ROOT"/deploy/kubernetes/*.yaml  "$TEMP_DIR/."
rm "$TEMP_DIR"/kustomization.yaml || true
cp "$COLLECTOR_REPO_ROOT/hack/test/deploy/memcached-config.yaml" "$TEMP_DIR/."
cp "$COLLECTOR_REPO_ROOT/hack/test/deploy/mysql-config.yaml" "$TEMP_DIR/."
cp "$COLLECTOR_REPO_ROOT/hack/test/deploy/prom-example.yaml" "$TEMP_DIR/."

pushd "$TEMP_DIR"
  kubectl config set-context --current --namespace="$NS"
  kubectl delete -f "$TEMP_DIR/."
  kubectl config set-context --current --namespace=default
popd
