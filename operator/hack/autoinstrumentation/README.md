# Kubernetes App Auto-Instrumentation via Pixie

These instructions will guide you through setting up auto-instrumentation of applications running on a Kubernetes cluster, through a "bring your own" [Pixie](https://docs.px.dev/about-pixie/what-is-pixie/) deployment.

By the end, you will see Application Maps in OpApps showing connections between microservices on a given Kubernetes cluster. You'll also get Request, Error, and Duration (RED) metrics between those connections.

> **Note**: The installation steps below require about 30 minutes to complete.

## Prerequisites

A Kubernetes cluster:
- Minimum of three nodes.
- Node VMs need a minimum of 4 vCPUs. For example, the GCP machine type of `e2-standard-4`.
- x86-64 CPU architecture. ARM is not supported.
- 600MB free memory per node to enabled Pixie data collection.

Refer to Pixie's [Setting up Kubernetes](https://docs.px.dev/installing-pixie/setting-up-k8s/) and [Requirements](https://docs.px.dev/installing-pixie/requirements/) documentation for more details.


## Install Operations for Applications Kubernetes Integration

**Note**: The steps below are a condensed version of the [README](/README.md#installation) for deploying the Operator into your Kubernetes cluster.

1. Install the Observability for Kubernetes Operator into the `observability-system` namespace.

   **Note**: If you already have the deprecated Kubernetes Integration installed by using Helm or manual deployment, *uninstall* it before you install the Operator.

   ```
   kubectl apply -f https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/deploy/wavefront-operator.yaml
   ```
2. Create a Kubernetes secret with your Wavefront API token.
   See [Managing API Tokens](https://docs.wavefront.com/wavefront_api.html#managing-api-tokens) page.
   ```
   kubectl create -n observability-system secret generic wavefront-secret --from-literal token=YOUR_WAVEFRONT_TOKEN
   ```
3. Create a `wavefront.yaml` file with your `Wavefront` Custom Resource configuration.
   ```yaml
   # Need to change YOUR_CLUSTER_NAME and YOUR_WAVEFRONT_URL
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

5. Deploy the agents with your configuration.
   ```
   kubectl apply -f wavefront.yaml
   ```

6. Run the following command to get status of the Kubernetes integration.
   ```
   kubectl get wavefront -n observability-system
   ```
   The command should return a table like the following, displaying Operator instance health:
   ```
   NAME        STATUS    PROXY           CLUSTER-COLLECTOR   NODE-COLLECTOR   LOGGING        AGE    MESSAGE
   wavefront   Healthy   Running (1/1)   Running (1/1)       Running (3/3)    Running (3/3)  2m4s   All components are healthy
   ```
   If `STATUS` is `Unhealthy`, check [troubleshooting](/docs/troubleshooting.md).

## Install Pixie

**Note:** the steps below are a condensed version of the [installation guide for Pixie Community Cloud](https://docs.px.dev/installing-pixie/install-guides/community-cloud-for-pixie/).

1. Sign up for a Pixie account at [https://work.withpixie.ai/](https://work.withpixie.ai/). **Note:** use `Sign-up With Google` to be in a shared org with other teammates on the same domain.


2. Install the Pixie CLI

   ```bash
   # Copy and run command to install the Pixie CLI.
   bash -c "$(curl -fsSL https://withpixie.ai/install.sh)"
   ```
   For alternate installation options, refer to the [Pixie CLI installation docs](https://docs.px.dev/installing-pixie/install-schemes/cli/).

3. Deploy Pixie
   
   Run the command below to install Pixie, being sure to change `YOUR_CLUSTER_NAME` to the same value that was set in `wavefront.yaml` above.

   ```bash
   # Set YOUR_CLUSTER_NAME to same value used in wavefront.yaml
   px deploy --pem_memory_limit=600Mi \
     --pem_flags="PL_TABLE_STORE_DATA_LIMIT_MB=150,PL_TABLE_STORE_HTTP_EVENTS_PERCENT=90,PL_STIRLING_SOURCES=kTracers" \
     --cluster_name=YOUR_CLUSTER_NAME
   ```

   > **Note**: The options above tune Pixie for the specific use-case of OpApps, and disable some features that are unused. This is done to reduce the memory required by Pixie per Kubernetes node to 600MB. The above configuration only enables `http_event` Pixie eBPF probes, and disables all others.

5. Check the status of the Pixie installation.

   ```
   kubectl -n pl describe vizier
   ```
   The `Status` section should report `Healthy` in the `Vizier Phase`:
   ```
   Status:
     Checksum:                        jX7VZmcDZXJQXtZY8wT9OogHz79z+nHGcEhhgqhY4/k=
     Last Reconciliation Phase Time:  2023-06-21T05:47:06Z
     Operator Version:                0.1.4+Distribution.01fedbe.20230620220143.1.jenkins
     Reconciliation Phase:            Ready
     Version:                         0.13.8
     Vizier Phase:                    Healthy
   ```

   If it isn't healthy, try looking for unhealthy pods and inspecting the logs:

   ```
   kubectl -n pl get pods
   ```

   An alternative for collecting logs is using the `px` CLI:

   ```
   px collect-logs
   ```

## Enable the OpenTelemetry Pixie Plugin

1. Navigate to the `Plugin` tab on the `Admin` page at https://work.withpixie.ai/admin/plugins.
2. Click the toggle to enable the OpenTelemetry plugin.
3. Expand the plugin row (with the arrow next to the toggle) and enter the export path of the Wavefront Proxy service deployed by the Operator: `wavefront-proxy.observability-system.svc.cluster.local:4317`.
4. Click the toggle to _disable_ "Secure connections with TLS" and press the SAVE button. The Wavefront Proxy does not support receiving OpenTelemetry data over TLS.


## Install the Operations for Applications Pixie Collection Script


1. Navigate to the `Data Retention Scripts` at https://work.withpixie.ai/configure-data-export.
2. The OpenTelemetry plugin comes with several pre-configured OTel export PxL scripts (Connection Stats, Network Stats, Resource Summary). Click the toggle to disable these scripts (if not already disabled). A custom Operations for Applications PxL script will be used to gather compatible instrumentation data.
3. Select the `+ CREATE SCRIPT` button.
4. Enter `Operations for Applications Spans (YOUR_CLUSTER_NAME)` in the `Script Name` field.
5. Select `OpenTelemetry` in the `Plugin` field.
6. Choose your cluster from the `Clusters` field.
7. Set the `Summary Window (Seconds)` field to `10`.
8. If the `Export URL` isn't already set to `wavefront-proxy.observability-system.svc.cluster.local:4317`, put that value in this field.
8. Replace the contents of the `PxL Script` field with the script at [/operator/hack/autoinstrumentation/spans.pxl](/operator/hack/autoinstrumentation/spans.pxl).
9. Click the `CREATE` button.
10. To validate that the data is being received by the Wavefront proxy, check logs for the the `wavefront-proxy` pod.

   ```bash
   kubectl logs deployment/wavefront-proxy -n observability-system | grep "Spans received rate:"
   ```

   If the plugin configuration was successful, you should see logs like:
   ```
   INFO  [AbstractReportableEntityHandler:printStats] [4317] Spans received rate: 0 sps (1 min), <1 sps (5 min), 0 sps (current).
   INFO  [AbstractReportableEntityHandler:printStats] [4317] Spans received rate: 6 sps (1 min), 2 sps (5 min), 0 sps (current).
   INFO  [AbstractReportableEntityHandler:printStats] [4317] Spans received rate: 6 sps (1 min), 1 sps (5 min), 0 sps (current).
   ```
   
   If the installation was unsuccessful, **or** the Kubernetes cluster doesn't have an active HTTP application workload, you may see three zeros:
   ```
   INFO  [AbstractReportableEntityHandler:printStats] [4317] Spans received rate: 0 sps (1 min), 0 sps (5 min), 0 sps (current).
   ```

## Finished!

If your Kubernetes cluster has an existing workload of HTTP traffic, go to https://YOUR_WAVEFRONT_URL/tracing/ and look for RED metric data to flow in.

The name of the Application in OpApps is determined by the Kubernetes namespace a service is deployed in. For example, if your cluster had two services named `frontend` and `backend` deployed in the namespace `demo-app`, you should find RED metrics for those services within a `demo-app`, based upon the HTTP traffic between them. Within a few minutes, you should also see connections between the edges of the services created.

If you need a demo application for your Kubernetes cluster, see the section below.

## Deploy a Demo Microservices App

If a demo HTTP workload is desired, the Pixie CLI can deploy one for you:

```shell
px demo deploy px-sock-shop
```

This demo application takes several minutes to stabilize after deployment.

To check the status of the application's pods, run:

```shell
kubectl get pods -n px-sock-shop
```

## Uninstalling

### Uninstall Pixie

To uninstall Pixie:
```shell
px delete
kubectl delete namespace olm
```

### Delete the Operations for Applications Pixie Script

Navigate to https://work.withpixie.ai/configure-data-export and either delete or disable the script named `Operations for Applications Spans (YOUR_CLUSTER_NAME)`.

### Uninstall Demo Microservices App

To uninstall the demo app:
```shell
px demo delete px-sock-shop
```

### Uninstalling Operations for Applications Kubernetes Integration

```shell
kubectl delete -f https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/deploy/wavefront-operator.yaml
```
