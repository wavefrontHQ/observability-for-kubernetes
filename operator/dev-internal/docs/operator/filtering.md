# Filtering

## Table of Contents
* [Introduction](#introduction)
* [Metrics Filtering](#metrics-filtering)

## Introduction
The Observability for Kubernetes Operator supports filtering metrics. Filters are based on [glob patterns](https://github.com/gobwas/glob#syntax) (similar to standard wildcards).

## Metrics Filtering

The Observability for Kubernetes Operator supports filtering metrics before they are reported to the Operations for Applications service. The following filtering options are supported for metrics:

  * **allowList**: List of glob patterns. Only metrics with names matching this list are reported.
  * **denyList**: List of glob patterns. Metrics with names matching this list are dropped.
  * **tagAllowList**: Map of tag names to list of glob patterns. Only metrics containing tag keys and values matching this list will be reported.
  * **tagDenyList**: Map of tag names to list of glob patterns. Metrics containing these tag keys and values will be dropped.
  * **tagInclude**: List of glob patterns. Tags with matching keys will be included. All other tags will be excluded.
  * **tagExclude**: List of glob patterns. Tags with matching keys will be excluded.
  * **tagGuaranteeList**: List of tag keys. Tags that are guaranteed to not be removed as part of limiting the point tags to the 20 tag limit. This list of tags keys will not be considered for filtering using `tagInclude` and `tagExclude` lists.

Filtering applies to all the metrics collected by the Kubernetes Metrics Collector:

```yaml
# Filters to apply towards all metrics collected.
filters:
# Filter out all go runtime metrics for kube-dns and apiserver.
denyList:
  - 'kube.dns.go.*'
  - 'kube.apiserver.go.*'

# Filter out metrics that have a namespace_name tag of kube-system.
tagDenyList:
  namespace_name:
    - 'kube-system'

# Filter out metrics that have a tag key of testing or begin with test.
tagExclude:
  - 'test*'

# Only allow metrics that start with the kubernetes prefix.
allowList:
  - 'kubernetes.*'

# Only allow metrics that have an environment tag of production or staging.
tagAllowList:
  env:
    - 'prod*'
    - 'staging*'

# Only allow metrics that have a tag key of cluster.
tagInclude:
  - 'cluster'

# Guarantee that if metrics have a point tag key "prod", the tag key will not be filtered out.
tagGuaranteeList:
  - 'prod'
```
