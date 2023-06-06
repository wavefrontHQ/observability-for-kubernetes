// Copyright 2019 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package stats provides internal metrics on the health of the Wavefront collector
package stats

import (
	"sync"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/filter"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"

	gometrics "github.com/rcrowley/go-metrics"
)

var doOnce sync.Once

type statsProvider struct {
	metrics.DefaultSourceProvider
	sources []metrics.Source
}

func (h *statsProvider) GetMetricsSources() []metrics.Source {
	return h.sources
}

func (h *statsProvider) Name() string {
	return "internal_stats_provider"
}

func NewInternalStatsProvider(cfg configuration.StatsSourceConfig) (metrics.SourceProvider, error) {
	if util.OnlyExportKubernetesEvents() {
		return &statsProvider{}, nil
	}

	prefix := configuration.GetStringValue(cfg.Prefix, "kubernetes.")
	tags := cfg.Tags
	filters := filter.FromConfig(cfg.Filters)

	src, err := newInternalMetricsSource(prefix, tags, filters)
	if err != nil {
		return nil, err
	}
	sources := make([]metrics.Source, 1)
	sources[0] = src

	doOnce.Do(func() { // Temporal solution for https://github.com/rcrowley/go-metrics/issues/252
		gometrics.RegisterRuntimeMemStats(gometrics.DefaultRegistry)
	})

	return &statsProvider{
		sources: sources,
	}, nil
}
