#!/bin/bash -e

REPO_ROOT="$(git rev-parse --show-toplevel)"
source "${REPO_ROOT}/scripts/k8s-utils.sh"
OPERATOR_DIR="${REPO_ROOT}/operator"

cd "$OPERATOR_DIR"

git config --global user.email "svc.wf-jenkins@vmware.com"
git config --global user.name "svc.wf-jenkins"
git remote set-url origin https://${TOKEN}@github.com/wavefronthq/observability-for-kubernetes.git

RELEASE_VERSION=$(get_next_operator_version)

git checkout .

VERSION=$RELEASE_VERSION$VERSION_POSTFIX

make copy-rbac-kustomization-yaml released-kubernetes-yaml
cp "${OPERATOR_DIR}"/dev-internal/deploy/wavefront-operator.yaml "${OPERATOR_DIR}"/build/wavefront-operator.yaml

current_version="$(get_next_collector_version)"
image_version="${current_version}${VERSION_POSTFIX}"

sed -i.bak "s%collector:.*$%collector: ${image_version}%" "${OPERATOR_DIR}"/build/wavefront-operator.yaml

# update rc branch
git checkout ../
git fetch
git checkout rc
git reset --hard origin/rc

git clean -dfx -e build
OPERATOR_FILE="wavefront-operator-${GIT_BRANCH}.yaml"
mv "${OPERATOR_DIR}"/build/wavefront-operator.yaml "$OPERATOR_FILE"

git add --all .
git commit -m "build $OPERATOR_FILE from $GIT_COMMIT" || exit 0
git push origin rc || exit 0
