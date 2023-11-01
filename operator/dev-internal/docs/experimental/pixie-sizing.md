# Configuring the Size of the Pixie Deployment

These instructions will guide you through configuring the size of your Pixie deployment. It covers [setting up the default sizing](#Configure Default Size), running the [Pixie Sizer tool](#Running the Pixie Sizer tool), and [validating that the chosen size is sufficient](#Validating the Chosen Size).

## Prerequisites

A Kubernetes cluster with Pixie enabled:
- For general guidance enabling Pixie, see this [README](/docs/experimental/autotracing.md).

## Configure Default Size

1. In order to begin sizing your cluster's Pixie deployment, you must begin by choosing one of several presets. These are:
   1. Small (recommended for clusters with TODO 3 nodes)
   1. Medium (recommended for clusters with TODO 50 nodes)
   1. Large (recommended for clusters with TODO 150 nodes)
   
2. Once you have chosen your preset size, specify it in the Wavefront CR yaml like this (selecting only one size):
```yaml
spec: 
  clusterSize: small | medium | large
```
   **Note**: If you are happy with your sizing, this is all that is required. The rest of this guide will cover how to further customize the resources, and how to check to see if the chosen size is appropriate.


## Customizing Resources

There are two main categories of settings that are able to be customized. They are the general Kubernetes pod resource requirements, and internal Pixie-specific memory settings.

1. Customizing Pod Resource Requirements
- The resource requirements for any workload can be customized as described in this Readme (TODO: write this readme)
- The specific Pixie workloads and their function are as follows:
  - kelvin
    - Query aggregator and exporter
  - vizier-pem
    - Data collection and short-term retention
  - vizier-metadata
    - Metadata cache used by Pixie
  - vizier-query-broker
    - Query scheduler
  - pl-nats
    - Internal Pixie message bus
- An example yaml snippet overriding the resource requirements for `vizier-pem`:
```yaml
spec: 
  #clusterSize will only be overridden for vizier-pem
  clusterSize: <small | medium | large>
  workloadResources: 
    vizier-pem: 
      requests: 
        cpu: 1
        memory: 1Gi
      limits: 
        cpu: 2
        memory: 2Gi
```

2. Customizing Pixie-specific Settings
- There are two values exposed by Pixie to customize its internal resource limits: `total_mib` and `http_events_percent`, which are set in the Wavefront CR as follows:
```yaml
spec:
  experimental: 
    pixie: 
        table_store_limits: 
          total_mib: 100
          http_events_percent: 20
```
- The `clusterSize` chosen will give a reasonable default for these settings, but they can be fine-tuned. Tuning these variables relies on knowledge of internal Pixie dynamics. The Pixie Sizer tool described in the next section will make a recommendation for what these values should be based on actual observed needs of the cluster.

## Running the Pixie Sizer tool

The Pixie Sizer is a deployment in the `observability-system` namespace that will monitor the state of the cluster and make recommendations for sizing adjustments. It should be installed by running the following commands:

```bash
# Make a working directory
mkdir /tmp/pixie-sizer
cd /tmp/pixie-sizer

# Download the install script
curl -O install.sh https://raw.githubusercontent.com/wavefrontHQ/observability-for-kubernetes/main/operator/pixie-sizer/install.sh
chmod +x install.sh

# Install the sizer into a cluster that has Pixie enabled
PS_TRAFFIC_SCALE_FACTOR=1.5 PS_SAMPLE_PERIOD_MINUTES=480 ./install.sh
```

The Pixie Sizer requires two inputs to function:
- `PS_SAMPLE_PERIOD_MINUTES` - The amount of time in minutes that the Sizer should observe before making a recommendation. A longer period should result in more traffic variance being included.
- `PS_TRAFFIC_SCALE_FACTOR` - The amount of extra capacity that the Sizer should account for to handle bursty traffic, expressed as a scalar. For example, a Scale Factor of 1.5 will leave enough excess capacity to handle a 50% increase in traffic over what was observed in the Sample Period.

In order to see the Sizer's recommendation, run:

```bash
kubectl --namespace observability-system logs --selector="app.kubernetes.io/component=pixie-sizer" --container=pixie-sizer --since=$PS_SAMPLE_PERIOD_MINUTES
```

## Validating the Chosen Size

 1. Checking for OOM Kills
   (TODO)
## Known Limitations