#!/bin/bash -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
OPERATOR_DIR="${REPO_ROOT}/operator"


RELEASE_VERSION=$(cat ./release/OPERATOR_VERSION)
NEW_VERSION=$(semver-cli inc patch "$RELEASE_VERSION")

VERSION=$NEW_VERSION$VERSION_POSTFIX make released-kubernetes-yaml

current_version="$(yq .data.collector "${OPERATOR_DIR}/config/manager/component_versions.yaml")"
bumped_version="$("${REPO_ROOT}"/scripts/get-bumped-version.sh -v "${current_version}" -s patch)"
image_version="${bumped_version}${VERSION_POSTFIX}"

sed -i.bak "s%collector:.*$%collector: ${image_version}%" "${OPERATOR_DIR}"/dev-internal/deploy/wavefront-operator.yaml
rm "${OPERATOR_DIR}"/dev-internal/deploy/wavefront-operator.yaml.bak

git add "${OPERATOR_DIR}"/dev-internal/deploy/wavefront-operator.yaml
git commit -m "build $OPERATOR_FILE from $GIT_COMMIT" || exit 0
git push || exit 0
