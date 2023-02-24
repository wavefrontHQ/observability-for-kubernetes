#!/bin/bash -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
source "${REPO_ROOT}/scripts/k8s-utils.sh"
OPERATOR_DIR="${REPO_ROOT}/operator"

git config --global user.email "svc.wf-jenkins@vmware.com"
git config --global user.name "svc.wf-jenkins"
git remote set-url origin https://${TOKEN}@github.com/wavefronthq/observability-for-kubernetes.git

RELEASE_VERSION=$(get_operator_version)
NEW_VERSION=$(semver-cli inc patch "$RELEASE_VERSION")

git checkout .

VERSION=$NEW_VERSION$VERSION_POSTFIX make released-kubernetes-yaml
cp "${OPERATOR_DIR}"/dev-internal/deploy/wavefront-operator.yaml "${OPERATOR_DIR}"/build/wavefront-operator.yaml

current_version="$(get_component_version collector)"
bumped_version="$("${REPO_ROOT}"/scripts/get-bumped-version.sh -v "${current_version}" -s minor)"
image_version="${bumped_version}${VERSION_POSTFIX}"

sed -i.bak "s%collector:.*$%collector: ${image_version}%" "${OPERATOR_DIR}"/build/wavefront-operator.yaml

git checkout .
git fetch
git checkout rc
git reset --hard origin/rc

git clean -dfx -e build
OPERATOR_FILE="wavefront-operator-${GIT_BRANCH}.yaml"
mv "${OPERATOR_DIR}"/build/wavefront-operator.yaml "$OPERATOR_FILE"

git add --all .
git commit -m "build $OPERATOR_FILE from $GIT_COMMIT" || exit 0
git push origin rc || exit 0
