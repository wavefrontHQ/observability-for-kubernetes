// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package configuration

import (
	"testing"

	"github.com/stretchr/testify/require"
)

const sampleFile = `
clusterName: new-collector
enableDiscovery: true
enableEvents: true
defaultCollectionInterval: 10s
omitBucketSuffix: true

experimental:
- histogram-conversion

sinks:
- proxyAddress: wavefront-proxy.default.svc.cluster.local:2878
  tags:
    env: gcp-dev
    image: 0.9.9-rc3
  filters:
    metricAllowList:
    - 'kubernetes.node.*'

    metricTagAllowList:
      nodename:
      - 'gke-vikramr-cluster*wj2d'

    tagInclude:
    - 'nodename'
- externalEndpointURL: 'https://example.com'
  type: external

events:
  filters:
    tagAllowList:
      namespace:
      - "default"
      component:
      - "pp"
    tagAllowListSets:
    - kind:
      - "Pod"
      reason:
      - "Scheduled"
    - kind:
      - "DaemonSet"
      reason:
      - "SuccessfulCreate"

sources:
  kubernetes_source:
    prefix: kubernetes.

  kubernetes_cadvisor_source:
    prefix: 'kubernetes.cadvisor.'
    filters:
      metricAllowList:
      - 'kubernetes.cadvisor.*'

  prometheus_sources:
  - url: 'https://kubernetes.default.svc.cluster.local:443'
    httpConfig:
      bearer_token_file: '/var/run/secrets/kubernetes.io/serviceaccount/token'
      tls_config:
        ca_file: '/var/run/secrets/kubernetes.io/serviceaccount/ca.crt'
        insecure_skip_verify: true
    prefix: 'kube.apiserver.'

  telegraf_sources:
    - plugins: [cpu]
      collection:
        interval: 1s
    - plugins: [mem]

discovery:
  annotation_excludes:
  - images:
    - 'not-redis:*'
    - '*not-redis*'
  plugins:
  - type: telegraf/redis
    name: "redis"
    selectors:
      images:
      - 'redis:*'
      - '*redis*'
    port: 6379
    scheme: "tcp"
    collection:
      interval: 1s
    conf: |
      servers = [${server}]
      password = bar
`

func TestFromYAML(t *testing.T) {
	cfg, err := FromYAML([]byte(sampleFile))
	if err != nil {
		t.Errorf("error loading yaml: %q", err)
		return
	}
	if len(cfg.Sinks) == 0 {
		t.Errorf("invalid sinks")
	}

	require.True(t, cfg.EnableEvents)
	require.Equal(t, "default", cfg.EventsConfig.Filters.TagAllowList["namespace"][0])
	require.Equal(t, "pp", cfg.EventsConfig.Filters.TagAllowList["component"][0])

	require.True(t, len(cfg.Sources.PrometheusConfigs) > 0)
	require.Equal(t, "kubernetes.", cfg.Sources.SummaryConfig.Prefix)
	require.Equal(t, "kube.apiserver.", cfg.Sources.PrometheusConfigs[0].Prefix)
	require.Equal(t, "kubernetes.cadvisor.", cfg.Sources.CadvisorConfig.Prefix)
	require.Equal(t, "histogram-conversion", cfg.Experimental[0])
	require.Equal(t, WavefrontSinkType, cfg.Sinks[0].Type)
	require.Equal(t, ExternalSinkType, cfg.Sinks[1].Type)
	require.Equal(t, "https://example.com", cfg.Sinks[1].ExternalEndpointURL)

	require.Equal(t, cfg.DiscoveryConfig.AnnotationExcludes[0].Images, []string{"not-redis:*", "*not-redis*"})
}
