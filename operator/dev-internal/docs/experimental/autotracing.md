# Kubernetes App Auto-Instrumentation via Pixie

These instructions will guide you through setting up auto-tracing of applications running on a Kubernetes cluster.

By the end, you will see Application Maps in OpApps showing connections between microservices on a given Kubernetes cluster. You'll also get Request, Error, and Duration (RED) metrics between those connections.

> **Note**: The installation steps below require about 10 minutes to complete.

## Prerequisites

A Kubernetes cluster:
- Minimum of three nodes.
- Node VMs need a minimum of 4 vCPUs. For example, the GCP machine type of `e2-standard-4`.
- x86-64 CPU architecture. ARM is not supported.
- 600MB free memory per node to enabled Pixie data collection.

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
     experimental:
       autotracing:
         enable: true
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

### Uninstall Demo Microservices App

To uninstall the demo app:
```shell
px demo delete px-sock-shop
```

### Uninstalling Operations for Applications Kubernetes Integration

```shell
kubectl delete -f https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/deploy/wavefront-operator.yaml
```

## Known Limitations

### App Map

Some edges may be missing from the app map. Edges should appear as long as:
* They represent HTTP traffic.
* They represent gRPC traffic and the server is a golang service compiled with debug symbols (the default for go).


