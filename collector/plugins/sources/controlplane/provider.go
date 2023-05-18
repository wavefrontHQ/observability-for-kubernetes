package controlplane

import (
	"fmt"
	"time"

	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources/prometheus"

	log "github.com/sirupsen/logrus"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/filter"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/httputil"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources/summary/kubelet"
)

const (
	metricsURL    = "https://kubernetes.default.svc/metrics"
	metricsSource = "control_plane_source"
	jitterTime    = time.Second * 40
	metricsPrefix = "kubernetes.controlplane."
)

type provider struct {
	metrics.DefaultSourceProvider

	providers []metrics.SourceProvider
}

func NewProvider(cfg configuration.ControlPlaneSourceConfig, summaryCfg configuration.SummarySourceConfig, client corev1.EndpointsGetter) (metrics.SourceProvider, error) {
	var providers []metrics.SourceProvider
	for _, promCfg := range buildPromConfigs(cfg, summaryCfg) {
		provider, err := prometheus.NewPrometheusProvider(promCfg, prometheus.InstancesFromEndpoints(client))
		if err != nil {
			return nil, fmt.Errorf("error building prometheus sources for control plane: %s", err.Error())
		}
		providers = append(providers, provider)
	}
	return &provider{providers: providers}, nil
}

func (p *provider) GetMetricsSources() []metrics.Source {
	var sources []metrics.Source
	for _, provider := range p.providers {
		sources = append(sources, provider.GetMetricsSources()...)
	}
	return sources
}

func (p *provider) Name() string {
	return metricsSource
}

func buildPromConfigs(cfg configuration.ControlPlaneSourceConfig, summaryCfg configuration.SummarySourceConfig) []configuration.PrometheusSourceConfig {
	var prometheusSourceConfigs []configuration.PrometheusSourceConfig

	kubeConfig, _, err := kubelet.GetKubeConfigs(summaryCfg)
	if err != nil {
		log.Infof("error %v", err)
		return nil
	}
	httpClientConfig := httputil.ClientConfig{
		BearerTokenFile: kubeConfig.BearerTokenFile,
		BearerToken:     kubeConfig.BearerToken,
		TLSConfig: httputil.TLSConfig{
			CAFile:             kubeConfig.CAFile,
			CertFile:           kubeConfig.CertFile,
			KeyFile:            kubeConfig.KeyFile,
			ServerName:         kubeConfig.ServerName,
			InsecureSkipVerify: kubeConfig.Insecure,
		},
	}
	metricAllowList := []string{
		metricsPrefix + "etcd.request.duration.seconds",
		metricsPrefix + "etcd.db.total.size.in.bytes.gauge",
		metricsPrefix + "workqueue.adds.total.counter",
		metricsPrefix + "workqueue.queue.duration.seconds",
	}

	promSourceConfig := createPrometheusSourceConfig("etcd-workqueue", httpClientConfig, metricAllowList, nil, cfg.Collection.Interval+jitterTime)
	prometheusSourceConfigs = append(prometheusSourceConfigs, promSourceConfig)

	apiServerAllowList := []string{
		metricsPrefix + "apiserver.request.duration.seconds",
		metricsPrefix + "apiserver.request.total.counter",
		metricsPrefix + "apiserver.storage.objects.gauge",
	}
	apiServerTagAllowList := map[string][]string{
		"resource": {"customresourcedefinitions", "namespaces", "lease", "nodes", "pods", "tokenreviews", "subjectaccessreviews"},
		"group":    {"authentication.k8s.io", "authorization.k8s.io", "certificates.k8s.io", "rbac.authorization.k8s.io"},
	}
	promApiServerSourceConfig := createPrometheusSourceConfig("apiserver", httpClientConfig, apiServerAllowList, apiServerTagAllowList, cfg.Collection.Interval)
	prometheusSourceConfigs = append(prometheusSourceConfigs, promApiServerSourceConfig)

	return prometheusSourceConfigs
}

func createPrometheusSourceConfig(name string, httpClientConfig httputil.ClientConfig, metricAllowList []string,
	metricTagAllowList map[string][]string, collectionInterval time.Duration) configuration.PrometheusSourceConfig {

	controlPlaneTransform := configuration.Transforms{
		Source: metricsSource,
		Prefix: metricsPrefix,
		Tags:   nil,
		Filters: filter.Config{
			MetricAllowList:    metricAllowList,
			MetricDenyList:     nil,
			MetricTagAllowList: metricTagAllowList,
			MetricTagDenyList:  nil,
			TagInclude:         nil,
			TagExclude:         nil,
		},
		ConvertHistograms: true,
	}

	sourceConfig := configuration.PrometheusSourceConfig{
		Transforms: controlPlaneTransform,
		Collection: configuration.CollectionConfig{
			Interval: collectionInterval,
			Timeout:  0,
		},
		URL:               metricsURL,
		HTTPClientConfig:  httpClientConfig,
		Discovered:        "",
		Name:              name,
		UseLeaderElection: true,
	}
	return sourceConfig
}
