# Manual Release Steps

```bash
#
# assume that we have committed the correct released versions for aria-integration and aria-configuration
#

# build
cd ~/workspace/observability-for-kubernetes

make -C operator -o kubernetes-yaml helm-kubernetes-yaml

cd helm-charts
rm aria-integration/templates/*.yaml.bak || true
rm aria-integration/Chart.yaml.bak || true

# publish to dev
helm package aria-integration
helm push aria-integration-*.tgz oci://projects.registry.vmware.com/tanzu_observability_keights_saas/helm-charts

helm package aria-configuration
helm push aria-configuration-*.tgz oci://projects.registry.vmware.com/tanzu_observability_keights_saas/helm-charts

# test
helm install aria-integration --create-namespace --namespace observability-system --version <VERSION> \
oci://projects.registry.vmware.com/tanzu_observability_keights_saas/helm-charts/aria-integration

helm install aria-configuration --namespace observability-system --version <VERSION> \
oci://projects.registry.vmware.com/tanzu_observability_keights_saas/helm-charts/aria-configuration \
--set clusterName=<YOUR CLUSTER NAME> \
--set k8sEvents.url=<YOUR EVENTS URL> \
--set k8sEvents.token=<YOUR LEMANS TOKEN>

# publish to prod
helm push aria-integration-*.tgz oci://projects.registry.vmware.com/tanzu_observability/helm-charts
helm push aria-configuration-*.tgz oci://projects.registry.vmware.com/tanzu_observability/helm-charts

# clean
rm aria-integration-*.tgz
rm aria-configuration-*.tgz
```
