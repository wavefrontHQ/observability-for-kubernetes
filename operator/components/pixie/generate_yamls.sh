#!/usr/bin/env bash
set -euo pipefail

REPO_ROOT=$(git rev-parse --show-toplevel)
rm -rf yamls

PIXIE_VERSION="0.14.7"

curl -L "https://github.com/pixie-io/pixie/releases/download/release%2Fvizier%2Fv${PIXIE_VERSION}/vizier_yamls.tar" --output yamls.tar

tar -xvf yamls.tar

pushd yamls

mkdir splits

# Split resources into their own yaml files
chmod +w vizier/vizier_metadata_persist_prod.yaml
chmod +w vizier_deps/nats_prod.yaml
echo -e "---\n$(cat vizier/vizier_metadata_persist_prod.yaml)" > vizier/vizier_metadata_persist_prod.yaml
echo -e "---\n$(cat vizier_deps/nats_prod.yaml)" > vizier_deps/nats_prod.yaml
files_to_apply=(vizier/vizier_metadata_persist_prod.yaml vizier/secrets.yaml vizier_deps/nats_prod.yaml)
cat "${files_to_apply[@]}" | csplit -n 3 -f 'splits/pixie-' - '/^---$/' "{$(($(cat "${files_to_apply[@]}" | grep -c '^\-\-\-$') - 2))}"

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

rm splits/secrets/01-secret-pl-deploy-secrets.yaml

rm splits/*cloud-conn*
rm splits/roles/*cloud-conn*

rm splits/03-serviceaccount-pl-updater-service-account.yaml
rm splits/roles/02-role-pl-updater-role.yaml
rm splits/roles/12-rolebinding-pl-updater-binding_pl-updater-role_pl-updater-service-account.yaml
rm splits/roles/20-clusterrolebinding-pl-updater-cluster-binding_pl-updater-cluster-role_pl-updater-service-account.yaml

rm splits/roles/03-role-pl-vizier-crd-role.yaml
rm splits/roles/13-rolebinding-pl-vizier-crd-binding_pl-vizier-crd-role_default.yaml
rm splits/roles/14-rolebinding-pl-vizier-crd-metadata-binding_pl-vizier-crd-role_metadata-service-account.yaml
rm splits/roles/17-rolebinding-pl-vizier-query-broker-crd-binding_pl-vizier-crd-role_query-broker-service-account.yaml

rm splits/roles/08-clusterrole-pl-updater-cluster-role.yaml
rm splits/roles/19-clusterrolebinding-pl-node-view-cluster-binding_pl-node-view_default.yaml

yq -i 'del( .spec.template.spec.initContainers[] | select(.name == "cc-wait") )' splits/12-deployment-kelvin.yaml
yq -i 'del( .spec.template.spec.initContainers[] | select(.name == "cc-wait") )' splits/14-deployment-vizier-query-broker.yaml
yq -i '(.spec.template.spec.containers[] | select(.name == "app") | .env) += {"name": "PL_CRON_SCRIPT_SOURCES", "value": "configmaps"}' splits/14-deployment-vizier-query-broker.yaml

git rm -rf "${REPO_ROOT}/operator/config/rbac/components/pixie/*.yaml" || true
mkdir -p "${REPO_ROOT}/operator/config/rbac/components/pixie"
cp splits/roles/*.yaml "${REPO_ROOT}/operator/config/rbac/components/pixie"

git rm -rf "${REPO_ROOT}/operator/components/pixie/*.yaml" || true
mkdir -p "${REPO_ROOT}/operator/components/pixie"
cp splits/secrets/*.yaml "${REPO_ROOT}/operator/components/pixie"
cp splits/*.yaml "${REPO_ROOT}/operator/components/pixie"

for f in "${REPO_ROOT}"/operator/config/rbac/components/pixie/*.yaml
do
  yq -i '.metadata.labels["app.kubernetes.io/name"] |= "wavefront"' "$f"
  yq -i '.metadata.labels["app.kubernetes.io/component"] |= "pixie"' "$f"
done

yq -i '.metadata.annotations["wavefront.com/conditionally-provision"] = "{{ .TLSCertsSecretExists }}"' "${REPO_ROOT}"/operator/components/pixie/12-deployment-kelvin.yaml
yq -i '.metadata.annotations["wavefront.com/conditionally-provision"] = "{{ .TLSCertsSecretExists }}"' "${REPO_ROOT}"/operator/components/pixie/14-deployment-vizier-query-broker.yaml
yq -i '.metadata.annotations["wavefront.com/conditionally-provision"] = "{{ .TLSCertsSecretExists }}"' "${REPO_ROOT}"/operator/components/pixie/15-statefulset-vizier-metadata.yaml
yq -i '.metadata.annotations["wavefront.com/conditionally-provision"] = "{{ .TLSCertsSecretExists }}"' "${REPO_ROOT}"/operator/components/pixie/16-daemonset-vizier-pem.yaml
yq -i '.metadata.annotations["wavefront.com/conditionally-provision"] = "{{ .TLSCertsSecretExists }}"' "${REPO_ROOT}"/operator/components/pixie/23-statefulset-pl-nats.yaml

yq -i '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_TABLE_STORE_DATA_LIMIT_MB", "value": "{{ .TableStoreLimits.TotalMiB }}"}' "${REPO_ROOT}/operator/components/pixie/16-daemonset-vizier-pem.yaml"
yq -i '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_TABLE_STORE_HTTP_EVENTS_PERCENT", "value": "{{ .TableStoreLimits.HttpEventsPercent }}"}' "${REPO_ROOT}/operator/components/pixie/16-daemonset-vizier-pem.yaml"
yq -i '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_TABLE_STORE_STIRLING_ERROR_LIMIT_BYTES", "value": "0"}' "${REPO_ROOT}/operator/components/pixie/16-daemonset-vizier-pem.yaml"
yq -i '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_TABLE_STORE_PROC_EXIT_EVENTS_LIMIT_BYTES", "value": "0"}' "${REPO_ROOT}/operator/components/pixie/16-daemonset-vizier-pem.yaml"
yq -i '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PX_STIRLING_HTTP_BODY_LIMIT_BYTES", "value": "{{ .MaxHTTPBodyBytes }}"}' "${REPO_ROOT}/operator/components/pixie/16-daemonset-vizier-pem.yaml"
yq -i '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_STIRLING_MAX_BODY_BYTES", "value": "{{ .MaxHTTPBodyBytes }}"}' "${REPO_ROOT}/operator/components/pixie/16-daemonset-vizier-pem.yaml"
yq -i '(.spec.template.spec.containers[] | select(.name == "pem") | .env) += {"name": "PL_STIRLING_SOURCES", "value": "{{ .StirlingSourcesEnv }}"}' "${REPO_ROOT}/operator/components/pixie/16-daemonset-vizier-pem.yaml"

sed -i 's%image: gcr.io/pixie-oss/pixie-dev-public/curl:multiarch-7.87.0%image: projects.registry.vmware.com/tanzu_observability/bitnami/os-shell:11%' "${REPO_ROOT}"/operator/components/pixie/*.yaml
sed -i 's%image: gcr.io%image: projects.registry.vmware.com/tanzu_observability%' "${REPO_ROOT}"/operator/components/pixie/*.yaml
sed -i 's/image: projects.registry.vmware.com\/tanzu_observability\/pixie-oss\/pixie-prod\/vizier-deps\/nats:2.9.19-scratch/image: projects.registry.vmware.com\/tanzu_observability\/pixie-oss\/pixie-prod\/vizier-deps\/nats:2.9.19-scratch/' "${REPO_ROOT}"/operator/components/pixie/23-statefulset-pl-nats.yaml
sed -i "s/:${PIXIE_VERSION}/:${PIXIE_VERSION}/" "${REPO_ROOT}"/operator/components/pixie/*.yaml
sed -i 's/@sha256:.*//' "${REPO_ROOT}"/operator/components/pixie/*.yaml
echo "  cluster-id: {{ .ClusterUUID }}" >> "${REPO_ROOT}/operator/components/pixie/00-secret-pl-cluster-secrets.yaml"
echo "  cluster-name: {{ .ClusterName }}" >> "${REPO_ROOT}/operator/components/pixie/00-secret-pl-cluster-secrets.yaml"
#sed -i "s/value: default/value: default\n{{- if (not .Experimental.Hub.Pixie.Enable) }}\n        - name: PL_TABLE_STORE_DATA_LIMIT_MB\n          value: \"150\"\n        - name: PL_TABLE_STORE_HTTP_EVENTS_PERCENT\n          value: \"90\"\n        - name: PL_STIRLING_SOURCES\n          value: \"kTracers\"\n{{- end }}/" "${REPO_ROOT}/operator/components/pixie/16-daemonset-vizier-pem.yaml"
sed -i 's/  PL_CLUSTER_NAME: ""/  PL_CLUSTER_NAME: {{ .ClusterName }}/' "${REPO_ROOT}/operator/components/pixie/18-configmap-pl-cloud-config.yaml"

# Iterate over files in the directory here as a workaround for yq interfering with namespace templatization instead of line 43
for file in "${REPO_ROOT}"/operator/components/pixie/*.yaml; do
    if [ -f "$file" ]; then
        sed -i 's/namespace: observability-system/namespace: {{ .Namespace }}/g' "$file"
    fi
done

git add "${REPO_ROOT}/operator/config/rbac/components/pixie"
git add "${REPO_ROOT}/operator/components/pixie"

popd

rm -rf yamls.tar yamls