// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package prometheus

import (
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/discovery"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources/prometheus"
)

func NewProviderInfo(handler metrics.ProviderHandler, prefix string) discovery.ProviderInfo {
	return discovery.ProviderInfo{
		Handler: handler,
		Factory: prometheus.NewFactory(),
		Encoder: newPrometheusEncoder(prefix),
	}
}
