# Deploy the Observability for Kubernetes Operator with a custom registry

Install the Observability for Kubernetes Operator into `observability-system` namespace.

**Note**: All the integration components use the same image registry in the Operator.

1. Copy the following images over to `YOUR_IMAGE_REGISTRY`, keeping the same repos and tags.


| Component | From | To |
|---|---|---|
| Observability for Kubernetes Operator | `projects.registry.vmware.com/tanzu_observability/kubernetes-operator:2.6.0` | `YOUR_IMAGE_REGISTRY/kubernetes-operator:2.6.0` |
| Kubernetes Metrics Collector | `projects.registry.vmware.com/tanzu_observability/kubernetes-collector:1.18.0` | `YOUR_IMAGE_REGISTRY/kubernetes-collector:1.18.0` |
| Wavefront Proxy | `projects.registry.vmware.com/tanzu_observability/proxy:12.4` | `YOUR_IMAGE_REGISTRY/proxy:12.4` |
| Operations for Applications logging | `projects.registry.vmware.com/tanzu_observability/kubernetes-operator-fluentbit:2.1.2` | `YOUR_IMAGE_REGISTRY/kubernetes-operator-fluentbit:2.1.2` |

2. Create a local directory called `observability`.
3. Download [wavefront-operator.yaml](https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/deploy/wavefront-operator.yaml) into the `observability` directory.
4. Create a `kustomization.yaml` file in the `observability` directory.
  ```yaml
  # Need to change YOUR_IMAGE_REGISTRY
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
   
  resources:
  - wavefront-operator.yaml
   
  images:
  - name: projects.registry.vmware.com/tanzu_observability/kubernetes-operator
    newName: YOUR_IMAGE_REGISTRY/kubernetes-operator
  ```
5. Deploy the Observability for Kubernetes Operator
  ```
  kubectl apply -k observability
  ```
6. Now follow the steps starting from step 2 in [Deploy the Kubernetes Metrics Collector and Wavefront Proxy with the Operator](../../README.md#Deploy-the-Kubernetes-Metrics-Collector-and-Wavefront-Proxy-with-the-Observability-for-Kubernetes-Operator)

# Deploy the Observability for Kubernetes Operator into a Custom Namespace

1. Create a local directory called `observability`.
2. Download [wavefront-operator.yaml](https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/deploy/wavefront-operator.yaml) into the `observability` directory.
3. Create a `kustomization.yaml` file in the `observability` directory.
  ```yaml
  # Need to change YOUR_NAMESPACE
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization

  resources:
  - wavefront-operator.yaml

  namespace: YOUR_NAMESPACE
  patches:
  - target:
       kind: RoleBinding
    patch: |-
       - op: replace
         path: /subjects/0/namespace
         value: YOUR_NAMESPACE
  - target:
       kind: ClusterRoleBinding
    patch: |-
       - op: replace
         path: /subjects/0/namespace
         value: YOUR_NAMESPACE
  ```
4. Deploy the Observability for Kubernetes Operator
  ```
  kubectl apply -k observability
  ```
5. Now follow the steps starting from step 2 in [Deploy the Kubernetes Metrics Collector and Wavefront Proxy with the Operator](../../README.md#Deploy-the-Kubernetes-Metrics-Collector-and-Wavefront-Proxy-with-the-Observability-for-Kubernetes-Operator),
   replacing `observability-system` with `YOUR_NAMESPACE`.
