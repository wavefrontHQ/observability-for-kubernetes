# Kubernetes Troubleshooting

Get help when you have problems with your Kubernetes setup. This page is divided into three core sections that correspond to the common setup issues that may occur as a result of integrating your Kubernetes cluster with Operations for Applications.

1. [No data flowing into Operations for Applications](#no-data-flowing-into-operations-for-applications)
2. [Missing or incomplete data flowing into Operations for Applications](#missing-or-incomplete-data-flowing-into-operations-for-applications)
3. [Running workloads not being discovered or monitored](#running-workloads-not-being-discovered-or-monitored)

For an in depth overview of the integration and how it is deployed, please refer to our [GitHub page](https://github.com/wavefrontHQ/observability-for-kubernetes).

## No Data Flowing into Operations for Applications

If you have identified that there is a problem with data flowing into Operations for Applications, please follow the steps below.

### Check Observability for Kubernetes Operator Status Locally
```
kubectl get wavefront -n observability-system
```
The result should look like this:
```
NAME        STATUS    PROXY           CLUSTER-COLLECTOR   NODE-COLLECTOR   LOGGING         AGE    MESSAGE
wavefront   Healthy   Running (1/1)   Running (1/1)       Running (3/3)    Running (3/3)   3h3m   All components are healthy
```

### Proxy Not Running or Unhealthy

The Wavefront proxy forwards logs, metrics, traces and spans from all other components to Operations for Applications. No data flowing into Operations for Applications means that the proxy might be failing.

Check the proxy logs for errors
```
kubectl logs deployment/wavefront-proxy -n observability-system | grep ERROR
```

### Common Proxy Log Errors

#### HTTP 401 Unauthorized
Run the following command to get your current token and follow resolution steps highlighted in [HTTP 401 Unauthorized ERROR Message](https://docs.wavefront.com/proxies_troubleshooting.html#proxy-error-messages)

Confirm that your Wavefront API token is correctly configured
```
kubectl get secrets wavefront-secret -n observability-system -o json | jq '.data' | cut -d '"' -f 4 | tr -d '{}' | base64 --decode
```

#### Unknown Host or Unable to Check In
- *Without an HTTP Proxy*

  - Check that you configured the correct `wavefrontUrl` in your `Wavefront` Custom Resource
    ```
    kubectl -n observability-system get wavefront -o=jsonpath='{.items[*].spec.wavefrontUrl}'
    ```
- *With an HTTP Proxy*

  - Verify that the proxy recognizes your HTTP proxy configuration
    ```
    kubectl logs deployment/wavefront-proxy -n observability-system | grep proxyHost
    ```
    The value after --proxyHost should match what you have configured as the http-url in your HTTP proxy secret

  - Determine the name of your HTTP proxy secret
    ```
    kubectl -n observability-system get wavefront -o=jsonpath='{.items[*].spec.dataExport.wavefrontProxy.httpProxy.secret}'
    ```
  - Verify that the secret has the proper keys and values, check out [our example](../deploy/scenarios/wavefront-proxy-with-http-proxy.yaml)
    ```
    kubectl -n observability-system get secret http-proxy-secret -o=json | jq -r '.data | to_entries[] | "echo \(.key|@sh) $(echo \(.value|@sh) | base64 --decode)"' | xargs -I{} sh -c {}
    ```
  - Check your HTTP proxy logs

### Cluster or Node Collector Not Running or Unhealthy

Check the logs for errors
```
kubectl logs deployment/wavefront-cluster-collector -n observability-system
kubectl logs daemonset/wavefront-node-collector -n observability-system
```

### Logging is Not Running or Unhealthy

Check the logs for errors
```
kubectl logs daemonset/wavefront-logging -n observability-system
```

## Missing or Incomplete Data Flowing Into Operations for Applications

If you are experiencing gaps in data, where expected metrics or metric tags are not showing, please review the following steps.

**Note:** For out of the box Kubernetes Control Plane dashboard, certain managed Kubernetes environments do not support collecting metrics of all control plane elements. For detailed information, please refer to our [supported metrics page](https://github.com/wavefrontHQ/observability-for-kubernetes/blob/main/docs/metrics.md#control-plane-metrics).

### Are All Components Healthy?
Check the Observability for Kubernetes Operator status locally to determine if all components are healthy by using the below command:
```
kubectl get wavefront -n observability-system
```
It will return a result that looks like this:
```
NAME        STATUS    PROXY           CLUSTER-COLLECTOR   NODE-COLLECTOR   LOGGING         AGE    MESSAGE
wavefront   Healthy   Running (1/1)   Running (1/1)       Running (3/3)    Running (3/3)   3h3m   All components are healthy
```
Verify that the `STATUS` column is `Healthy` and all the requested resources are running i.e., `Running (1/1)`.

### Is There a Proxy Backlog?

You can check if the proxy is having backlog issues by following the instructions on [406 - Cannot Post Push Data WARN Message](https://docs.wavefront.com/proxies_troubleshooting.html#proxy-warn-messages). 
If your proxy is having backlog issues, below are some options for fixes:
- Filter more metrics - Refer [this example scenario](../deploy/scenarios/wavefront-collector-filtering.yaml) for filtering metrics
- Increase limits - Contact VMware Aria Operations for Applications Support to request a higher backend limit as suggested in [406 - Cannot Post Push Data WARN Message](https://docs.wavefront.com/proxies_troubleshooting.html#proxy-warn-messages)

### Are Metrics Being Dropped?

Formerly, we would see the following error in the proxy logs when a metric has too many tags: `Too many point tags`.
However, logic has been added to the collector to automatically drop tags in priority order
to ensure that metrics make it through to the proxy and no longer cause this error.
This is the order of the logic used to drop tags:
1. Explicitly excluded tags (from `tagExclude` config).
   Refer [here](../deploy/scenarios/wavefront-full-config.yaml) for an example scenario.
1. Tags are empty or are interpreted to be empty (`"tag.key": ""`, `"tag.key": "-"`, or `"tag.key": "/"`).
1. Tags are explicitly excluded
   (`"namespace_id": "..."`, `"host_id": "..."`, `"pod_id": "..."`, or `"hostname": "..."`).
1. Tag **values** are duplicated, and the shorter key is kept
   (`"tag.key": "same value"` is kept instead of `"tag.super.long.key": "same value"`).
1. Tag key matches `alpha.*` or `beta.*`, after keys have been sorted
   (e.g. `"alpha.eksctl.io/nodegroup-name": "arm-group"` or `"beta.kubernetes.io/arch": "amd64"`).
1. (As a last resort) tag key matches IaaS-specific tags, after keys have been sorted
   (`"kubernetes.azure.com/agentpool": "agentpool"`, `"topology.gke.io/zone": "us-central1-c"`, `"eksctl.io/nodegroup-name": "arm-group"`, etc.).

### Are Metrics Being Filtered?

Check custom resource configuration for verifying the metrics that are being filtered, by running the below command.
```
kubectl describe wavefront -n observability-system
```
If you would like to customize the metrics being filtered, refer [here](../deploy/scenarios/wavefront-collector-filtering.yaml) for an example scenario.

### Is my Custom Resource Config File Configured Correctly

Check for unhealthy status by running below command.
```
kubectl get wavefront -n observability-system
```
If there are any configuration or validation errors, the `MESSAGE` column in the result will describe the error.

## Running Workloads Not Being Discovered or Monitored

- Check the Kubernetes Metrics Collector Troubleshooting dashboard in the Kubernetes integration for collection errors. The `Collection Errors per Type` chart and `Collection Errors per Endpoint` chart can be used to find the sources whose metrics are not being collected
- Refer to [this](../deploy/scenarios/wavefront-full-config.yaml) example scenario for configuring sources for metric collection
- Check the cluster collector logs to verify if the source was configured for the metrics to be collected
  ```
  kubectl logs deployment/wavefront-cluster-collector -n observability-system
  ```