// Copyright 2018-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/experimental"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/version"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sinks/factory"

	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/leadership"

	log "github.com/sirupsen/logrus"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/agent"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/configuration"
	kube_config "github.com/wavefronthq/observability-for-kubernetes/collector/internal/kubernetes"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/metrics"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/options"
	"github.com/wavefronthq/observability-for-kubernetes/collector/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/discovery"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/events"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/manager"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/processors"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sinks"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources"
	"github.com/wavefronthq/observability-for-kubernetes/collector/plugins/sources/summary"

	kube_client "k8s.io/client-go/kubernetes"
	v1listers "k8s.io/client-go/listers/core/v1"
)

var discWatcher util.FileWatcher

func main() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetLevel(log.InfoLevel)
	log.SetOutput(os.Stdout)

	opt := options.Parse()

	if opt.Version {
		fmt.Println(fmt.Sprintf("version: %s\ncommit: %s", version.Version, version.Commit))
		os.Exit(0)
	}

	switch opt.LogLevel {
	case "trace":
		log.SetLevel(log.TraceLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	}

	log.Infof(strings.Join(os.Args, " "))
	log.Infof("wavefront-collector version %v", version.Version)
	enableProfiling(opt.EnableProfiling)
	enableForcedGC(opt.ForceGC)

	preRegister(opt)
	cfg := configuration.LoadOrDie(opt.ConfigFile)
	cfg = convertOrDie(opt, cfg)
	ag := createAgentOrDie(cfg)
	registerListeners(ag, opt)
	waitForStop()
}

func preRegister(opt *options.CollectorRunOptions) {
	util.SetAgentType(opt.AgentType)
	if util.GetNodeName() == "" && util.ScrapeOnlyOwnNode() {
		log.Fatalf("missing environment variable %s", util.NodeNameEnvVar)
	}

	setMaxProcs(opt)
	version.RegisterMetric()
}

func createAgentOrDie(cfg *configuration.Config) *agent.Agent {
	experimental.DisableAll()
	for _, feature := range cfg.Experimental {
		experimental.EnableFeature(feature)
	}

	// backwards compat: used by prometheus sources to format histogram metric names
	setEnvVar("omitBucketSuffix", strconv.FormatBool(cfg.OmitBucketSuffix))

	clusterName := cfg.ClusterName

	kubeClient := createKubeClientOrDie(*cfg.Sources.SummaryConfig)
	workloadCache := getWorkloadCacheOrDie(kubeClient)
	if cfg.Sources.StateConfig != nil {
		cfg.Sources.StateConfig.KubeClient = kubeClient
		cfg.Sources.StateConfig.WorkloadCache = workloadCache
	}

	// create sources manager
	sourceManager := sources.Manager()
	sourceManager.SetClient(kubeClient)
	sourceManager.SetDefaultCollectionInterval(cfg.DefaultCollectionInterval)
	err := sourceManager.BuildProviders(*cfg.Sources)
	if err != nil {
		log.Fatalf("Failed to create source manager: %v", err)
	}

	// Create sink managers. Must be done before creating event router.
	sinkManager := createSinkManagerOrDie(cfg.Sinks, cfg.SinkExportDataTimeout)

	// Events
	var eventRouter *events.EventRouter
	if cfg.EventsAreEnabled() {
		events.Log.Info("Events collection enabled")
		eventRouter = events.NewEventRouter(kubeClient, cfg.EventsConfig, sinkManager, util.ScrapeCluster(), workloadCache)
	} else {
		events.Log.Info("Events collection disabled")
	}

	podLister := getPodListerOrDie(kubeClient)

	discoveryManager := createDiscoveryManagerOrDie(kubeClient, cfg, sourceManager, podLister)
	dataProcessors := createDataProcessorsOrDie(kubeClient, clusterName, podLister, workloadCache, cfg)
	flushManager, err := manager.NewFlushManager(dataProcessors, sinkManager, cfg.FlushInterval)
	if err != nil {
		log.Fatalf("Failed to create main manager: %v", err)
	}

	// start leader-election
	if util.ScrapeCluster() {
		_, err = leadership.Subscribe(kubeClient, "agent")
	}
	if err != nil {
		log.Fatalf("Failed to start leader election: %v", err)
	}

	// create and start agent
	ag := agent.NewAgent(flushManager, discoveryManager, eventRouter)
	ag.Start()
	return ag
}

func registerListeners(ag *agent.Agent, opt *options.CollectorRunOptions) {
	handler := &reloader{ag: ag}
	if opt.ConfigFile != "" {
		listener := configuration.NewFileListener(handler)
		watcher := util.NewFileWatcher(opt.ConfigFile, listener, 30*time.Second)
		watcher.Watch()
	}
}

func createDiscoveryManagerOrDie(
	client *kube_client.Clientset,
	cfg *configuration.Config,
	handler metrics.ProviderHandler,
	podLister v1listers.PodLister,
) *discovery.Manager {
	if cfg.EnableDiscovery {
		serviceLister := getServiceListerOrDie(client)
		nodeLister := getNodeListerOrDie(client)

		return discovery.NewDiscoveryManager(discovery.RunConfig{
			KubeClient:      client,
			DiscoveryConfig: cfg.DiscoveryConfig,
			Handler:         handler,
			Lister:          discovery.NewResourceLister(podLister, serviceLister, nodeLister),
			ScrapeCluster:   util.ScrapeCluster(),
		})
	}
	return nil
}

func createSinkManagerOrDie(cfgs []*configuration.SinkConfig, sinkExportDataTimeout time.Duration) sinks.Sink {
	sinksFactory := factory.NewSinkFactory()
	sinkList := sinksFactory.BuildAll(cfgs)

	for _, sink := range sinkList {
		log.Infof("Starting with %s", sink.Name())
	}
	sinkManager, err := sinks.NewSinkManager(sinkList, sinkExportDataTimeout, sinks.DefaultSinkStopTimeout)
	if err != nil {
		log.Fatalf("Failed to create sink manager: %v", err)
	}
	return sinkManager
}

func getPodListerOrDie(kubeClient *kube_client.Clientset) v1listers.PodLister {
	podLister, err := util.GetPodLister(kubeClient)
	if err != nil {
		log.Fatalf("Failed to create podLister: %v", err)
	}
	return podLister
}

func getWorkloadCacheOrDie(kubeClient *kube_client.Clientset) util.WorkloadCache {
	workloadCache, err := util.GetWorkloadCache(kubeClient)
	if err != nil {
		log.Fatalf("Failed to initialize workload cache: %v", err)
	}
	return workloadCache
}

func createKubeClientOrDie(cfg configuration.SummarySourceConfig) *kube_client.Clientset {
	kubeConfig, err := kube_config.GetKubeClientConfig(cfg)
	if err != nil {
		log.Fatalf("Failed to get client config: %v", err)
	}
	return kube_client.NewForConfigOrDie(kubeConfig)
}

func createDataProcessorsOrDie(kubeClient *kube_client.Clientset, cluster string, podLister v1listers.PodLister, workloadCache util.WorkloadCache, cfg *configuration.Config) []metrics.Processor {
	labelCopier, err := util.NewLabelCopier(",", []string{}, []string{})
	if err != nil {
		log.Fatalf("Failed to initialize label copier: %v", err)
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
		log.Fatalf("Failed to create NamespaceBasedEnricher: %v", err)
	}
	dataProcessors = append(dataProcessors, namespaceBasedEnricher)

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

	metricsToSkip := metricsToAggregate
	dataProcessors = append(dataProcessors,
		processors.NewPodResourceAggregator(podLister),
		processors.NewPodAggregator(metricsToSkip),
		processors.NewNamespaceAggregator(metricsToAggregate),
		processors.NewNodeAggregator(metricsToAggregateForNode),
		processors.NewClusterAggregator(metricsToAggregate),
	)

	nodeAutoscalingEnricher, err := processors.NewNodeAutoscalingEnricher(kubeClient, labelCopier)
	if err != nil {
		log.Fatalf("Failed to create NodeAutoscalingEnricher: %v", err)
	}
	dataProcessors = append(dataProcessors, nodeAutoscalingEnricher)

	// this always needs to be the last processor
	wavefrontCoverter, err := summary.NewPointConverter(*cfg.Sources.SummaryConfig, cluster)
	if err != nil {
		log.Fatalf("Failed to create WavefrontPointConverter: %v", err)
	}
	dataProcessors = append(dataProcessors, wavefrontCoverter)

	return dataProcessors
}

func calculateCollectionInterval(cfg *configuration.Config) time.Duration {
	collectionInterval := cfg.DefaultCollectionInterval
	if cfg.Sources.SummaryConfig.Collection.Interval > 0 {
		collectionInterval = cfg.Sources.SummaryConfig.Collection.Interval
	}
	return collectionInterval
}

func getServiceListerOrDie(kubeClient *kube_client.Clientset) v1listers.ServiceLister {
	serviceLister, err := util.GetServiceLister(kubeClient)
	if err != nil {
		log.Fatalf("Failed to create serviceLister: %v", err)
	}
	return serviceLister
}

func getNodeListerOrDie(kubeClient *kube_client.Clientset) v1listers.NodeLister {
	nodeLister, _, err := util.GetNodeLister(kubeClient)
	if err != nil {
		log.Fatalf("Failed to create nodeLister: %v", err)
	}
	return nodeLister
}

func setMaxProcs(opt *options.CollectorRunOptions) {
	// Allow as many threads as we have cores unless the user specified a value.
	var numProcs int
	if opt.MaxProcs < 1 {
		numProcs = runtime.NumCPU()
		if numProcs == 1 {
			// default to 2
			numProcs = 2
		}
	} else {
		numProcs = opt.MaxProcs
	}
	runtime.GOMAXPROCS(numProcs)

	// Check if the setting was successful.
	actualNumProcs := runtime.GOMAXPROCS(0)
	if actualNumProcs != numProcs {
		log.Warningf("Specified max procs of %d but using %d", numProcs, actualNumProcs)
	}
}

func enableProfiling(enable bool) {
	if enable {
		go func() {
			log.Info("Starting pprof server at: http://localhost:9090/debug/pprof")
			if err := http.ListenAndServe("localhost:9090", nil); err != nil {
				log.Errorf("E! %v", err)
			}
		}()
	}
}

func enableForcedGC(enable bool) {
	if enable {
		log.Info("enabling forced garbage collection")
		setEnvVar(util.ForceGC, "true")
	}
}

func setEnvVar(key, val string) {
	err := os.Setenv(key, val)
	if err != nil {
		log.Errorf("error setting environment variable %s: %v", key, err)
	}
}

func waitForStop() {
	select {}
}

type reloader struct {
	mtx sync.Mutex
	ag  *agent.Agent
	opt *options.CollectorRunOptions
}

// Handles changes to collector or discovery configuration
func (r *reloader) Handle(cfg interface{}) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	switch cfg.(type) {
	case *configuration.Config:
		r.handleCollectorCfg(cfg.(*configuration.Config))
	}
}

func (r *reloader) handleCollectorCfg(cfg *configuration.Config) {
	log.Infof("collector configuration changed")

	// stop the previous agent and start a new agent
	r.ag.Stop()
	r.ag = createAgentOrDie(cfg)
}

// converts flags to configuration for backwards compatibility support
func convertOrDie(opt *options.CollectorRunOptions, cfg *configuration.Config) *configuration.Config {
	// omit flags if config file is provided
	if cfg != nil {
		log.Info("using configuration file, omitting flags")
		return cfg
	}
	optsCfg, err := opt.Convert()
	if err != nil {
		log.Fatalf("error converting flags to config: %v", err)
	}
	return optsCfg
}
