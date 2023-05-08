#!/usr/bin/env bash
set -e

rm -rf pixie_yamls

px deploy --extract_yaml . --deploy_key $PX_DEPLOY_KEY --use_etcd_operator=true

tar -xvf yamls.tar

pushd pixie_yamls

mkdir splits

# Split resources into their own yaml files
files_to_apply=(00_secrets.yaml 01_nats.yaml 02_etcd.yaml 03_vizier_etcd.yaml)
cat "${files_to_apply[@]}" | csplit -n 3 -f 'splits/pixie-' - '/^---$/' "{$(($(cat "${files_to_apply[@]}" | grep -c '^\-\-\-$') - 2))}"

# Remove duplicate resources
fdupes -f splits | grep -v '^$' | xargs rm -v

# rename everything to a yaml file
original_file_names=($(echo splits/pixie-*))
mkdir -p splits/roles
mkdir -p splits/secrets
for index in "${!original_file_names[@]}"; do
  original_file_name="${original_file_names[$index]}"
  kind="$(grep '^kind:' "$original_file_name" | cut -d':' -f2 | xargs | tr '[:upper:]' '[:lower:]')"
  name="$(grep '^  name:' "$original_file_name" | cut -d':' -f2  | xargs)"
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

popd