# Helm Charts for Aria Operator

**NOTE:** These charts are not for use outside of Aria Hub. They will not work for standalone Wavefront/OpApps customers.

## Installing

**Helm 3.8+**

Helm 3.8 adopted a different distribution model for Charts. OCI image registries are the preferred method of
distributing Charts going forward.

```
helm install aria-operator oci://projects.registry.vmware.com/tanzu_observability/helm-charts/aria-operator

helm install aria-hub oci://projects.registry.vmware.com/tanzu_observability/helm-charts/aria-hub \
  --set clusterName=<CLUSTER_NAME> \
  --set k8sEvents.url=<URL> \
  --set k8sEvents.token=<LEMANS_TOKEN>
```

## Uninstalling

```
helm uninstall aria-hub 

helm uninstall aria-operator
```

## Developing

To package and push a helm chart to Harbor Dev:
```
helm package ./aria-operator
helm push aria-operator-*.tgz oci://projects.registry.vmware.com/tanzu_observability_keights_saas/helm-charts

helm package ./aria-hub
helm push aria-hub-*.tgz oci://projects.registry.vmware.com/tanzu_observability_keights_saas/helm-charts
```

To package and push a helm chart to Harbor Prod:
```
helm package ./aria-operator
helm push aria-operator-*.tgz oci://projects.registry.vmware.com/tanzu_observability/helm-charts

helm package ./aria-hub
helm push aria-hub-*.tgz oci://projects.registry.vmware.com/tanzu_observability/helm-charts
```

