# Helm Feature Comparison
Observability for Kubernetes Operator feature comparison with [Helm install](https://github.com/wavefrontHQ/helm/tree/master/wavefront#configuration):


| Helm Collector Parameter | Observability for Kubernetes Operator Custom Resource `spec`. | Description |
|---|---|---|
| `clusterName` | `clusterName` | ClusterName is a unique name for the Kubernetes cluster to be identified via a metric tag on Operations for Applications. |
| `wavefront.url` | `wavefrontUrl` | The URL of your product cluster. Ex: https://<your_cluster>.wavefront.com. |
| `wavefront.token` | `wavefrontTokenSecret` | WavefrontTokenSecret is the name of the secret that contains a Wavefront API Token. |
| `collector.enabled` | `dataCollection.metrics.enable` | Metrics holds the configuration for node and cluster collectors. |
| `collector.image.repository` | `Not currently supported` | Kubernetes Metrics Collector image registry and name. |
| `collector.image.tag` | `Not currently supported` | Kubernetes Metrics Collector image tag. |
| `collector.image.pullPolicy` | `Not currently supported` | Kubernetes Metrics Collector image pull policy. |
| `collector.image.updateStrategy` | `Not currently supported` | Kubernetes Metrics Collector updateStrategy. |
| `collector.useDaemonset` | `NA` | The operator uses an improved leader Deployment/DaemonSet architecture. |
| `collector.maxProx` | `NA` | Not supported in Helm or Operator install. |
| `collector.logLevel` | `Not currently supported` | Min logging level (info, debug, trace). |
| `collector.interval` | `dataExport.metrics.defaultCollectionInterval` | Default metrics collection interval. |
| `collector.flushInterval` | `NA` | This option was removed and the optimal flush interval is automatically set. |
| `collector.sinkDelay` | `NA` | This option was removed. |
| `collector.useReadOnlyPort` | `NA` | This option was removed. |
| `collector.useProxy` | `Not currently supported` | The Operator no longer supports direct ingestion. It requires internal or external Wavefront proxy configuration. |
| `collector.proxyAddress` | `dataExport.externalWavefrontProxy.Url` | The Wavefront proxy URL that the Collector sends metrics to. |
| `collector.apiServerMerics` | `Can enable with custom collector config` | Collect metrics about Kubernetes API server. |
| `collector.controlplane.enabled` | `Can disable with custom collector config` | Enable control plane metrics. |
| `collector.controlplane. collection.interval` | `Can be configured with custom collector config` | The collection interval for the control plane metrics. |
| `collector.kubernetesState` | `Can disable with custom collector config` | Collect metrics about Kubernetes resource states. |
| `collector.collector.cadvisor. enable` | `Can enable with custom collector config` | Enable cAdvisor Prometheus endpoint. See the cAdvisor docs for details on what metrics are available. |
| `collector.filters.metricDenyList` | `dataCollection.metrics.filters.denyList` | List of metric patterns to deny. |
| `collector.filters.metricAllowList` | `dataCollection.metrics.filters.allowList` | List of metric patterns to allow. |
| `collector.tags` | `dataCollection.metrics.tags` | Map of tags (key/value) to add to all metrics collected. |
| `collector.discovery.enabled` | `dataCollection.metrics.enableDiscovery` | Rules based and Prometheus endpoints auto-discovery. Defaults to true. |
| `collector.discovery. annotationPrefix` | `Can configure with custom collector config` | Replaces prometheus.io as prefix for annotations of auto-discovered Prometheus endpoints. |
| `collector.discovery. enableRuntimeConfigs` | `Can configure with custom collector config` | Enable runtime discovery rules. |
| `collector.discovery. annotationExcludes` | `Can configure with custom collector config` | Exclude resources from annotation based auto-discovery. |
| `collector.discovery.config` | `Can configure with custom collector config` | Exclude resources from annotation based auto-discovery. |
| `collector.resources` | `dataCollection.metrics.nodeCollector.resources` `dataCollection.metrics.clusterCollector.resources` | Configuration for rules based auto-discovery. |
| `imagePullSecrets` | `imagePullSecret` | Enable Wavefront proxy and Kubernetes Metrics Collector to pull from private image repositories. **Note:** Secret must exist in namespace that will be used for the installation. Currently, the operator supports a single imagePullSecret.|
| `proxy.enabled` | `dataExport.wavefrontProxy.enable` | Whether to enable the Wavefront proxy. Defaults to true. Disable to use `dataExport.externalWavefrontProxy.Url`. |
| `proxy.image.repository` | `Not currently supported` | Kubernetes Metrics Collector image registry and name. |
| `proxy.image.tag` | `Not currently supported` | Wavefront proxy image tag. |
| `proxy.image.pullPolicy` | `Not currently supported` | Wavefront proxy image pull policy. |
| `proxy.replicas` | `Not currently supported` | Replicas to deploy for Wavefront proxy (usually 1). |
| `proxy.port` | `dataExport.wavefrontProxy.metricPort` | MetricPort is the port for sending Wavefront data format metrics. Defaults to 2878. |
| `proxy.httpProxyHost` | `dataExport.wavefrontProxy.httpProxy.secret` | Name of the secret containing the HttpProxy configuration. |
| `proxy.httpProxyPort` | `dataExport.wavefrontProxy.httpProxy.secret` | Name of the secret containing the HttpProxy configuration. |
| `proxy.useHttpProxyCAcert` | `dataExport.wavefrontProxy.httpProxy.secret` | Name of the secret containing the HttpProxy configuration. |
| `proxy.httpProxyUser` | `dataExport.wavefrontProxy.httpProxy.secret` | Name of the secret containing the HttpProxy configuration. |
| `proxy.httpProxyPassword` | `dataExport.wavefrontProxy.httpProxy.secret` | Name of the secret containing the HttpProxy configuration. |
| `proxy.tracePort` | `dataExport.wavefrontProxy.tracing.wavefront.port` | Port for sending distributed Operations for Applications format tracing data (usually 30000). |
| `proxy.jaegerPort` | `dataExport.wavefrontProxy.tracing.jaeger.port` | Port for Jaeger format tracing data (usually 30001). |
| `proxy. traceJaegerHttpListenerPort` | `dataExport.wavefrontProxy.tracing.jaeger. httpPort` | HttpPort for Jaeger Thrift format data (usually 30080). |
| `proxy. traceJaegerGrpcListenerPort` | `dataExport.wavefrontProxy.tracing.jaeger. grpcPort` | GrpcPort for Jaeger gRPC format data (usually 14250). |
| `proxy.zipkinPort` | `dataExport.wavefrontProxy.tracing.zipkin.port` | Port for Zipkin format tracing data (usually 9411). |
| `proxy.traceSamplingRate` | `dataExport.wavefrontProxy.tracing.wavefront. samplingRate` | Distributed tracing data sampling rate (0 to 1). |
| `proxy.traceSamplingDuration` | `dataExport.wavefrontProxy.tracing.wavefront. samplingDuration` | When set to greater than 0, spans that exceed this duration will force trace to be sampled (ms). |
| `proxy. traceJaegerApplicationName` | `dataExport.wavefrontProxy.tracing.jaeger. applicationName` | Custom application name for traces received on Jaeger's HTTP or gRPC port. |
| `proxy. traceZipkinApplicationName` | `dataExport.wavefrontProxy.tracing.zipkin. applicationName` | Custom application name for traces received on Zipkin's port. |
| `proxy.histogramPort` | `dataExport.wavefrontProxy.histogram.port` | Port for Operations for Applications histogram distributions (usually 40000). |
| `proxy.histogramMinutePort` | `dataExport.wavefrontProxy.histogram.minutePort` | Port to accumulate 1-minute based histograms in the Operations for Applications data format (usually 40001). |
| `proxy.histogramHourPort` | `dataExport.wavefrontProxy.histogram.hourPort` | Port to accumulate 1-hour based histograms in the Operations for Applications data format (usually 40002). |
| `proxy.histogramDayPort` | `dataExport.wavefrontProxy.histogram.dayPort` | Port to accumulate 1-day based histograms in the Operations for Applications data format (usually 40002). |
| `proxy.deltaCounterPort` | `dataExport.wavefrontProxy.deltaCounterPort` | Port to send delta counters on Operations for Applications data format (usually 50000). |
| `proxy.args` | `dataExport.wavefrontProxy.args` | Additional Wavefront proxy properties can be passed as command line arguments in the `--<property_name> <value>` format. Multiple properties can be specified. |
| `proxy.preprocessor.rules.yaml` | `dataExport.wavefrontProxy.preprocessor` | Name of the configmap containing a rules.yaml key with proxy preprocessing rules. |
| `proxy.heap` | `Not currently supported` | Wavefront proxy Java heap maximum usage (java -Xmx command line option). |
| `proxy.preprocessor.rules.yaml` | `Configurable via custom preproccesor rules` | YAML configuraiton for Wavefront proxy preprocessor rules. |
| `rbac.create` | `Not currently supported` | Create RBAC resources. |
| `serviceAccount.create` | `Not currently supported` | Create an Operations for Applications service account. |
| `serviceAccount.name` | `Not currently supported` | Name of the Operations for Applications service account. |
| `kubeStateMetrics.enabled` | `Not currently supported` | Set up and enable Kube-State-Metrics for collection. |
| `vspheretanzu.enabled` | `Not currently supported` | Enable and create role binding for vSphere with Tanzu kubernetes cluster. |
