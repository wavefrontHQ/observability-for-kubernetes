# Deploy the Observability for Kubernetes Operator with a custom registry

Install the Observability for Kubernetes Operator into `observability-system` namespace.

**Note**: All the integration components use the same image registry in the Operator.

1. Copy the following images over to `YOUR_IMAGE_REGISTRY`, keeping the same repos and tags.


| Component | From | To |
|---|---|---|
| Observability for Kubernetes Operator | `caapm/kubernetes-operator:2.32.0` | `YOUR_IMAGE_REGISTRY/kubernetes-operator:2.32.0` |
| Kubernetes Metrics Collector | `caapm/kubernetes-collector:1.44.0` | `YOUR_IMAGE_REGISTRY/kubernetes-collector:1.44.0` |
| Wavefront Proxy | `caapm/proxy:13.9` | `YOUR_IMAGE_REGISTRY/proxy:13.9` |
| Operations for Applications logging | `projects.registry.vmware.com/tanzu_observability/kubernetes-operator-fluentbit:2.2.0` | `YOUR_IMAGE_REGISTRY/kubernetes-operator-fluentbit:2.2.0` |

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
5. If your image registry needs authentication, create an image registry secret in the same namespace as the operator (The default namespace is `observability-system`) by following steps [here](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/), then modify the `kustomization.yaml` to include your image registry secret. 
  ```yaml
  # Need to change YOUR_IMAGE_REGISTRY and YOUR_IMAGE_REGISTRY_SECRET
  apiVersion: kustomize.config.k8s.io/v1beta1
  kind: Kustomization
 
  resources:
  - wavefront-operator.yaml
 
  images:
  - name: projects.registry.vmware.com/tanzu_observability/kubernetes-operator
    newName: YOUR_IMAGE_REGISTRY/kubernetes-operator

  patches:
  - target:
        kind: Deployment
        name: wavefront-controller-manager
    patch: |-
        - op: add
          path: /spec/template/spec/imagePullSecrets
          value:
          - name: YOUR_IMAGE_REGISTRY_SECRET
  ```
6. Deploy the Observability for Kubernetes Operator
  ```
  kubectl apply -k observability
  ```
7. Now follow the steps starting from step 2 in [Deploy the Kubernetes Metrics Collector and Wavefront Proxy with the Operator](../../README.md#Deploy-the-Kubernetes-Metrics-Collector-and-Wavefront-Proxy-with-the-Observability-for-Kubernetes-Operator). Also, add your image registry secret to the Wavefront Custom Resource as shown in this [example](../../deploy/scenarios/wavefront-custom-private-registry.yaml).

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
