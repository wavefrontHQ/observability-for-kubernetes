# K8s App Zero-Instrumentation via Pixie

These instructions will guide you through setting up zero-instrumentation of a Kubernetes cluster via
Pixie technology.

## Prerequisites

GKE is the only supported Kuberenetes provider right now. In addition, your cluster needs to conform to the following:

- Minimum of five nodes.
- Node VMs need a minimum of 4 vCPUs (`e2-standard-4`).


## Install Operations for Applications Kubernetes Integration

Follow Steps 1 and 2 in the [README](/README.md#installation) to deploy the Operator into your Kubernetes cluster.

For Step 3, create a `wavefront.yaml` file with your Wavefront Custom Resource configuration. The
simplest configuration to enable collection of Pixie data is:

```yaml
# Need to change YOUR_CLUSTER_NAME and YOUR_WAVEFRONT_URL accordingly
apiVersion: wavefront.com/v1alpha1
kind: Wavefront
metadata:
  name: wavefront
  namespace: observability-system
spec:
  clusterName: YOUR_CLUSTER_NAME
  wavefrontUrl: YOUR_WAVEFRONT_URL
  dataCollection:
    metrics:
      enable: true
  dataExport:
    wavefrontProxy:
      enable: true
      otlp:
        grpcPort: 4317
        resourceAttrsOnMetricsIncluded: true
```

Continue following the installation through Step 6 to deploy the Collector and Proxy with the `wavefront.yaml`
above. Note that Step 4 (logging beta) is optional and not required for Pixie data collection.


## Install Pixie

### 1. Sign up

Visit the [product page](https://work.withpixie.ai/) and sign up.

**Note:** use `Sign-up With Google` to be in a shared org with other teammates on the same domain.


### 2. Install the Pixie CLI

```bash
# Copy and run command to install the Pixie CLI.
bash -c "$(curl -fsSL https://withpixie.ai/install.sh)"
```

For alternate installation options, refer to the [Pixie CLI installation docs](https://docs.px.dev/installing-pixie/install-schemes/cli/).

### 3. Deploy Pixie

Pixie uses Operator Lifecycle Manager, and it is assumed your Kubernetes cluster does not have an
existing OLM installed.

It is recommended to deploy Pixie with a memory limit of no more than 25% of your Kubernetes node's
total memory. So if you have 4Gi total memory on your nodes, you'll want to use a 1Gi memory limit.

The lowest recommended value is 1Gi. 1Gi is not a suitable limit for a cluster with high throughput,
but may be suitable for a small cluster with limited resources.

```bash
# List Pixie deployment options.
px deploy --help

# Deploy the Pixie Platform in your K8s cluster (No OLM present on cluster).
px deploy

# Deploy Pixie with a specific memory limit (2Gi is the default, 1Gi is the minimum recommended)
px deploy --pem_memory_limit=2.5Gi
```

Pixie deploys the following pods to your cluster. Note that the number of `vizier-pem` pods 
correlates with the number of nodes in your cluster, so your  deployment may contain more PEM pods.

```bash
NAMESPACE           NAME
olm                 catalog-operator
olm                 olm-operator
pl                  kelvin
pl                  nats-operator
pl                  pl-nats-1
pl                  vizier-certmgr
pl                  vizier-cloud-connector
pl                  vizier-metadata
pl                  vizier-pem
pl                  vizier-pem
pl                  vizier-proxy
pl                  vizier-query-broker
px-operator         77003c9dbf251055f0bb3e36308fe05d818164208a466a15d27acfddeejt7tq
px-operator         pixie-operator-index
px-operator         vizier-operator
```


## Deploy a Demo Microservices App

```shell
px demo deploy px-sock-shop
```

This demo application takes several minutes to stabilize after deployment.

To check the status of the application's pods, run:

```shell
kubectl get pods -n px-sock-shop
```

## Enable the OpenTelemetry Pixie Plugin

1. Navigate to the `Plugin` tab on the `Admin` page at https://work.withpixie.ai/admin/plugins.
2. Click the toggle to enable the OpenTelemetry plugin.
3. Expand the plugin row (with the arrow next to the toggle) and enter the export path of the Wavefront Proxy service deployed by the Operator: `wavefront-proxy.observability-system.svc.cluster.local:4317`.
4. Click the toggle to disable "Secure connections with TLS" and press the SAVE button. The Wavefront Proxy does not support receiving OpenTelemetry data over TLS.

## Install the Operations for Applications Pixie Script

1. Click the database icon in the left nav bar to open the [data export configuration](https://work.withpixie.ai/configure-data-export) page.
2. The OpenTelemetry plugin comes with several pre-configured OTel export PxL scripts. Click the toggle to disable these scripts. A custom Operations for Applications PxL script will be used to gather compatible instrumentation data.
3. Select the `+ CREATE SCRIPT` button.
4. Enter `Operations for Applications Spans` in the `Script Name` field.
5. Select `OpenTelemetry` in the `Plugin` field.
6. Choose your cluster from the `Clusters` field.
7. Set the `Summary Window (Seconds)` field to `60`.
8. Replace the contents of the `PxL` field with the script at [/operator/hack/autoinstrumentation/spans.pxl](/operator/hack/autoinstrumentation/spans.pxl).
9. Click the `CREATE` button.
10. To validate that the data is being received by the Wavefront proxy, check logs for the the `wavefront-proxy-*` pod. If the plugin configuration was successful, you should see logs: 
```
2022-04-15T21:17:27.530Z    INFO    loggingexporter/logging_exporter.go:54    MetricsExporter    {"#metrics": 32}
2022-04-15T21:17:37.570Z    INFO    loggingexporter/logging_exporter.go:54    MetricsExporter    {"#metrics": 30}
2022-04-15T21:17:47.609Z    INFO    loggingexporter/logging_exporter.go:54    MetricsExporter    {"#metrics": 29}
2022-04-15T21:17:57.449Z    INFO    loggingexporter/logging_exporter.go:54    MetricsExporter    {"#metrics": 29}
```