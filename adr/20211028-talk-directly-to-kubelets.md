# 20211028 Talk to kubelets directly instead of going through the apiserver proxy

## Context

The initial implementation of the new cAdvisor source was calling the metrics/cadvisor endpoint through a node proxy endpoint. By doing this, each request to get cAdvisor data was going through the [k8s apiserver proxy](https://kubernetes.io/docs/concepts/cluster-administration/proxies/) and back into the node / kubelet.

A later design review brought up that the fact the summary source was going directly to kubelet vs going through the node proxy and that this implementation could be a performance concern for larger k8s clusters.

## Decision

Follow the pattern of summary source and call directly to kubelet whenever possible. Move calculation of the kubelet URL to KubeleteClientConfig (this is how we're sharing behavior between the cAdvisor and summary sources)

## Status
[Implemented](https://github.com/wavefrontHQ/observability-for-kubernetes/commit/0f071a3cab79f516cde38b7aabc3fe92598b3256)

## Consequences
Increases code complexity slightly since we needed extract / share the calculation of the kubelet URL, but overall performance and scalability should be better. In addition, we now have a standard pattern to call directly to the kubelet.
