#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
rm -rf yamls

curl -L "https://github.com/pixie-io/pixie/releases/download/release%2Fvizier%2Fv0.14.4/vizier_yamls.tar" --output yamls.tar

tar -xvf yamls.tar

pushd yamls

mkdir splits

# Split resources into their own yaml files
chmod +w vizier/vizier_metadata_persist_prod.yaml
chmod +w vizier_deps/nats_prod.yaml
echo -e "---\n$(cat vizier/vizier_metadata_persist_prod.yaml)" > vizier/vizier_metadata_persist_prod.yaml
echo -e "---\n$(cat vizier_deps/nats_prod.yaml)" > vizier_deps/nats_prod.yaml
files_to_apply=(vizier/vizier_metadata_persist_prod.yaml vizier/secrets.yaml vizier_deps/nats_prod.yaml)
cat "${files_to_apply[@]}" | csplit -n 3 -f 'splits/autoinstrumentation-' - '/^---$/' "{$(($(cat "${files_to_apply[@]}" | grep -c '^\-\-\-$') - 2))}"

# Remove duplicate resources
#duplicates=$(fdupes -f splits)
#if [[ $duplicates != "" ]]; then
#  echo "$duplicates" | grep -v '^$' | xargs rm -fv
#fi

# rename everything to a yaml file
original_file_names=($(echo splits/autoinstrumentation-*))
mkdir -p splits/roles
mkdir -p splits/secrets
for index in "${!original_file_names[@]}"; do
  original_file_name="${original_file_names[$index]}"
  kind="$(grep '^kind:' "$original_file_name" | cut -d':' -f2 | xargs | tr '[:upper:]' '[:lower:]')"
  name="$(grep '^  name:' "$original_file_name" | cut -d':' -f2  | xargs | tr '[:blank:]' '_')"
  if [[ "$kind" =~ role ]]; then
    num="$(printf "%02d" "$(find splits/roles -maxdepth 1 -name '*.yaml' | wc -l | xargs)")"
    new_file="splits/roles/$num-$kind-$name.yaml"
  elif [[ "$kind" =~ secret ]]; then
    num="$(printf "%02d" "$(find splits/secrets -maxdepth 1 -name '*.yaml' | wc -l | xargs)")"
    new_file="splits/secrets/$num-$kind-$name.yaml"
  else
    num="$(printf "%02d" "$(find splits -maxdepth 1 -name '*.yaml' | wc -l | xargs)")"
    new_file="splits/$num-$kind-$name.yaml"
  fi
  sed 's/namespace: pl/namespace: observability-system/g' "$original_file_name" | sed '1d' > "$new_file"
  rm "$original_file_name"
done

rm splits/*cloud-conn*
rm splits/secrets/01-secret-pl-deploy-secrets.yaml
rm splits/roles/*cloud-conn*

yq -i 'del( .spec.template.spec.initContainers[] | select(.name == "cc-wait") )' splits/12-deployment-kelvin.yaml
yq -i 'del( .spec.template.spec.initContainers[] | select(.name == "cc-wait") )' splits/14-deployment-vizier-query-broker.yaml
yq -i '(.spec.template.spec.containers[] | select(.name == "app") | .env) += {"name": "PL_CRON_SCRIPT_SOURCES", "value": "configmaps"}' splits/14-deployment-vizier-query-broker.yaml

git rm -rf "${REPO_ROOT}/operator/config/rbac/components/autoinstrumentation/*.yaml"
mkdir -p "${REPO_ROOT}/operator/config/rbac/components/autoinstrumentation"
cp splits/roles/*.yaml "${REPO_ROOT}/operator/config/rbac/components/autoinstrumentation"
git add "${REPO_ROOT}/operator/config/rbac/components/autoinstrumentation"

git rm -rf "${REPO_ROOT}/operator/deploy/internal/autoinstrumentation/*.yaml"
mkdir -p "${REPO_ROOT}/operator/deploy/internal/autoinstrumentation"
cp splits/secrets/*.yaml "${REPO_ROOT}/operator/deploy/internal/autoinstrumentation"
cp splits/*.yaml "${REPO_ROOT}/operator/deploy/internal/autoinstrumentation"

#find "${REPO_ROOT}"/operator/deploy/internal/autoinstrumentation/ -type f -name '*.yaml' -exec sed -i '' 's/          image: gcr.io/          image: projects.registry.vmware.com\/asap/' {}
sed -i '' 's/image: gcr.io/image: projects.registry.vmware.com\/asap/' "${REPO_ROOT}"/operator/deploy/internal/autoinstrumentation/*.yaml
sed -i '' 's/@sha256:.*//' "${REPO_ROOT}"/operator/deploy/internal/autoinstrumentation/*.yaml
sed -i '' 's/nats:2.9.19//' "${REPO_ROOT}"/operator/deploy/internal/autoinstrumentation/*.yaml
sed -i '' 's/  PL_CLUSTER_NAME: ""/  PL_CLUSTER_NAME: {{ .ClusterName }}/' "${REPO_ROOT}/operator/deploy/internal/autoinstrumentation/18-configmap-pl-cloud-config.yaml"
echo "  cluster-id: {{ .ClusterUUID }}" >> "${REPO_ROOT}/operator/deploy/internal/autoinstrumentation/00-secret-pl-cluster-secrets.yaml"
echo "  cluster-name: {{ .ClusterName }}" >> "${REPO_ROOT}/operator/deploy/internal/autoinstrumentation/00-secret-pl-cluster-secrets.yaml"

git add "${REPO_ROOT}/operator/deploy/internal/autoinstrumentation"

popd

rm -rf yamls.tar yamls