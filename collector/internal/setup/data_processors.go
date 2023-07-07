package setup

import (
	"fmt"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/processors"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources/summary"
	kube_client "k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/listers/core/v1"
)

func CreateDataProcessors(kubeClient kube_client.Interface, cluster string, podLister v1.PodLister, workloadCache util.WorkloadCache, cfg *configuration.Config) ([]metrics.Processor, error) {
	labelCopier, err := util.NewLabelCopier(",", []string{}, []string{})
	if err != nil {
		return nil, fmt.Errorf("Failed to initialize label copier: %v", err)
	}

	dataProcessors := []metrics.Processor{
		processors.NewRateCalculator(metrics.RateMetricsMapping),
		processors.NewDistributionRateCalculator(),
		processors.NewCumulativeDistributionConverter(),
	}

	collectionInterval := calculateCollectionInterval(cfg)
	podBasedEnricher := processors.NewPodBasedEnricher(podLister, workloadCache, labelCopier, collectionInterval)
	dataProcessors = append(dataProcessors, podBasedEnricher)

	namespaceBasedEnricher, err := processors.NewNamespaceBasedEnricher(kubeClient)
	if err != nil {
		return nil, fmt.Errorf("Failed to create NamespaceBasedEnricher: %v", err)
	}
	dataProcessors = append(dataProcessors, namespaceBasedEnricher)

	dataProcessors = append(dataProcessors, createMetricAggregators(podLister)...)

	nodeAutoscalingEnricher, err := processors.NewNodeAutoscalingEnricher(kubeClient, labelCopier)
	if err != nil {
		return nil, fmt.Errorf("Failed to create NodeAutoscalingEnricher: %v", err)
	}
	dataProcessors = append(dataProcessors, nodeAutoscalingEnricher)

	// this always needs to be the last processor
	wavefrontCoverter, err := summary.NewPointConverter(*cfg.Sources.SummaryConfig, cluster)
	if err != nil {
		return nil, fmt.Errorf("Failed to create WavefrontPointConverter: %v", err)
	}
	dataProcessors = append(dataProcessors, wavefrontCoverter)

	return dataProcessors, nil
}

func calculateCollectionInterval(cfg *configuration.Config) time.Duration {
	collectionInterval := cfg.DefaultCollectionInterval
	if cfg.Sources.SummaryConfig.Collection.Interval > 0 {
		collectionInterval = cfg.Sources.SummaryConfig.Collection.Interval
	}
	return collectionInterval
}

func createMetricAggregators(podLister v1.PodLister) []metrics.Processor {
	// Note: Only change which metrics are aggregated if you know what effect it will have.

	// Although the Request/Limit metrics are aggregated by the Pod Resource Aggregator,
	// we need to still aggregate them to the Namespace and Cluster levels.
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

	// These specific metrics are aggregated by the Pod Resource Aggregator,
	// which utilizes the kubectl code for aggregation, so we don't want
	// to re-aggregate them from the container level using the Pod Aggregator.
	podMetricsToNotAggregate := []string{
		metrics.MetricCpuRequest.Name,
		metrics.MetricCpuLimit.Name,
		metrics.MetricMemoryRequest.Name,
		metrics.MetricMemoryLimit.Name,
	}

	return []metrics.Processor{
		processors.NewPodResourceAggregator(podLister),
		processors.NewPodAggregator(podMetricsToNotAggregate),
		processors.NewNamespaceAggregator(metricsToAggregate),
		processors.NewNodeAggregator(metricsToAggregateForNode),
		processors.NewClusterAggregator(metricsToAggregate),
	}
}
