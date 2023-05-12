#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
rm -rf pixie_yamls

px deploy --extract_yaml . --deploy_key replace_me --use_etcd_operator=false --cluster_name=replace_me

tar -xvf yamls.tar

pushd pixie_yamls

mkdir splits

# Split resources into their own yaml files
files_to_apply=(00_secrets.yaml 01_nats.yaml 04_vizier_persistent.yaml)
cat "${files_to_apply[@]}" | csplit -n 3 -f 'splits/pixie-' - '/^---$/' "{$(($(cat "${files_to_apply[@]}" | grep -c '^\-\-\-$') - 2))}"

# Remove duplicate resources
duplicates=$(fdupes -f splits)
if [[ $duplicates != "" ]]; then
  echo "$duplicates" | grep -v '^$' | xargs rm -fv
fi

# rename everything to a yaml file
original_file_names=($(echo splits/pixie-*))
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

git rm -rf "${REPO_ROOT}/operator/config/rbac/components/pixie/*.yaml"
mkdir -p "${REPO_ROOT}/operator/config/rbac/components/pixie"
cp splits/roles/*.yaml "${REPO_ROOT}/operator/config/rbac/components/pixie"
git add "${REPO_ROOT}/operator/config/rbac/components/pixie"

git rm -rf "${REPO_ROOT}/operator/deploy/internal/pixie/*.yaml"
mkdir -p "${REPO_ROOT}/operator/deploy/internal/pixie"
cp splits/secrets/*.yaml "${REPO_ROOT}/operator/deploy/internal/pixie"
cp splits/*.yaml "${REPO_ROOT}/operator/deploy/internal/pixie"

sed -i '' 's/  PL_CLUSTER_NAME: "replace_me"/  PL_CLUSTER_NAME: {{ .ClusterName }}/' "${REPO_ROOT}/operator/deploy/internal/pixie/00-configmap-pl-cloud-config.yaml"
sed -i '' 's/  deploy-key: "replace_me"/  deploy-key: {{ .Experimental.AutoInstrumentation.DeployKey }}/' "${REPO_ROOT}/operator/deploy/internal/pixie/01-secret-pl-deploy-secrets.yaml"
git add "${REPO_ROOT}/operator/deploy/internal/pixie"

popd

rm -rf yamls.tar pixie_yamls