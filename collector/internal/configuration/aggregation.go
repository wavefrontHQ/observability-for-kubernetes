package configuration

import (
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/processors"
	"k8s.io/client-go/listers/core/v1"
)

func CreateMetricAggregators(dataProcessors []metrics.Processor, podLister v1.PodLister) []metrics.Processor {
	metricsToAggregate := []string{
		metrics.MetricCpuUsageRate.Name,
		metrics.MetricMemoryUsage.Name,
		metrics.MetricCpuRequest.Name,
		metrics.MetricCpuLimit.Name,
		metrics.MetricMemoryRequest.Name,
		metrics.MetricMemoryLimit.Name,
	}

	metricsToAggregateForNode := []string{
		metrics.MetricCpuRequest.Name,
		metrics.MetricCpuLimit.Name,
		metrics.MetricMemoryRequest.Name,
		metrics.MetricMemoryLimit.Name,
		metrics.MetricEphemeralStorageRequest.Name,
		metrics.MetricEphemeralStorageLimit.Name,
	}

	podMetricsToNotAggregate := []string{
		metrics.MetricCpuRequest.Name,
		metrics.MetricCpuLimit.Name,
		metrics.MetricMemoryRequest.Name,
		metrics.MetricMemoryLimit.Name,
	}

	dataProcessors = append(dataProcessors,
		processors.NewPodResourceAggregator(podLister),
		processors.NewPodAggregator(podMetricsToNotAggregate),
		processors.NewNamespaceAggregator(metricsToAggregate),
		processors.NewNodeAggregator(metricsToAggregateForNode),
		processors.NewClusterAggregator(metricsToAggregate),
	)
	return dataProcessors
}
