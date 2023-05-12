# Tuning Pixie/Vizier PEM Memory Usage

## Context

Pixie recommends of minimum of 1GB memory allocated to Vizier PEM pods, but we'd like to reduce that if possible when deploying to customer K8s clusters.

We were attempting to achieve a stable K8s environment. **Stable** was defined as:
1. No `OOMKilled` restarts for `vizier-pem` DaemonSet pods
2. No indications of lost data due to lack of memory. Specifically, no RowBatch error logs from `vizier-pem` such as:
  > `pem E20230502 20:27:57.663512 1453658 source_connector.cc:64] Failed to push data. Message = RowBatch size (61179) is bigger than maximum table size (-898779). │`

Our test environments were configured as:
- GKE cluster of 3 nodes (e2-standard-4)
- Single `spans.pxl` script running every 10 seconds, generating OTel spans from http_events data
- Synthetic user workloads of px-sock-shop and otel demo app deployed
- Testing a minimum of 24 hours for a given config


The findings below are a synthesis from [K8SSAAS-1783](https://jira.eng.vmware.com/browse/K8SSAAS-1783). Prior memory tuning ticket is [K8SSAAS-1691](https://jira.eng.vmware.com/browse/K8SSAAS-1691).

## Decision

Given the above environment configuration, the deployment configuration that was most stable:

```shell
px deploy --cluster_name=my-gke-cluster --pem_memory_limit=600Mi \
--pem_flags="PL_TABLE_STORE_DATA_LIMIT_MB=150,PL_TABLE_STORE_HTTP_EVENTS_PERCENT=90,PL_STIRLING_SOURCES=kTracers"
```
Results:
- Zero `OOMKilled`.
- Zero RowBatch size errors.

We are incorporating a PEM memory limit of 600Mi, and the above `pem_flags` into our deployment of Vizier onto customer K8s clusters.

The biggest takeaway we found is the `PL_STIRLING_SOURCES` config, which lets you narrow the number of probes running on each PEM and get lower memory usage. Our PxL script only uses http_events, and so we only need to enable the `kTracers` stirling source.

Related to only using http_events, we can dedicate the majority of the table store to them via `PL_TABLE_STORE_HTTP_EVENTS_PERCENT`.

##  Other Options Considered

### Configuration #1

`px deploy --cluster_name=goppegard-gke --pem_memory_limit=500Mi # (operator set a default PL_TABLE_STORE_DATA_LIMIT_MB=300)`

Results:
- Every 20 minutes 2 of the 3 pods were `OOMKilled`.


### Configuration #2

`px deploy --cluster_name=goppegard-gke --pem_memory_limit=500Mi --pem_flags="PL_TABLE_STORE_DATA_LIMIT_MB=150,PL_TABLE_STORE_HTTP_EVENTS_PERCENT=90,PL_STIRLING_SOURCES=kTracers"`

Results:
- Zero `OOMKilled`.
- RowBatch size errors:
  > `vizier-pem-ldl6g pem E20230505 19:02:28.884579 115127 source_connector.cc:64] Failed to push data. Message = RowBatch size (2500668) is bigger than maximum table size (1348169).`

  

## Status

We are incorporating our configuration into [K8SSAAS-1844](https://jira.eng.vmware.com/browse/K8SSAAS-1844).

A lot of it still feels like a black box. Based on some threads in the Pixie Slack (see below), it sounds like we’d need to do memory profiling of the PEM processes for more detail.

Pixie Slack Threads:
- https://pixie-community.slack.com/archives/C03TC7D3N67/p1683063935563699
- https://pixie-community.slack.com/archives/C03TC7D3N67/p1681856929956449