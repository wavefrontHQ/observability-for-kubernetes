package collector

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

func FromWavefront(cr *wf.Wavefront) ComponentConfig {
	config := ComponentConfig{
		// required
		Enable:                    (cr.Spec.CanExportData && cr.Spec.DataCollection.Metrics.Enable || cr.Spec.Experimental.Insights.Enable),
		MetricsEnable:             cr.Spec.CanExportData && cr.Spec.DataCollection.Metrics.Enable,
		ShouldValidate:            cr.Spec.DataCollection.Metrics.Enable || cr.Spec.Experimental.Insights.Enable,
		CustomConfig:              cr.Spec.DataCollection.Metrics.CustomConfig,
		ControllerManagerUID:      cr.Spec.ControllerManagerUID,
		ClusterName:               cr.Spec.ClusterName,
		ClusterUUID:               cr.Spec.ClusterUUID,
		EnableDiscovery:           cr.Spec.DataCollection.Metrics.EnableDiscovery,
		DefaultCollectionInterval: cr.Spec.DataCollection.Metrics.DefaultCollectionInterval,
		ProxyAddress:              cr.Spec.DataCollection.Metrics.ProxyAddress,
		Namespace:                 cr.Spec.Namespace,
		ProxyAvailableReplicas:    cr.Spec.DataExport.WavefrontProxy.AvailableReplicas,
		ImageRegistry:             cr.Spec.ImageRegistry,
		CollectorVersion:          cr.Spec.DataCollection.Metrics.CollectorVersion,
		ClusterCollectorResources: cr.Spec.DataCollection.Metrics.ClusterCollector.Resources,
		NodeCollectorResources:    cr.Spec.DataCollection.Metrics.NodeCollector.Resources,
		CollectorConfigName:       cr.Spec.DataCollection.Metrics.CollectorConfigName,

		// optional
		Filters: cr.Spec.DataCollection.Metrics.Filters,
		Tags:    cr.Spec.DataCollection.Metrics.Tags,
		KubernetesEvents: KubernetesEvents{
			Enable:         cr.Spec.Experimental.Insights.Enable,
			IngestionUrl:   cr.Spec.Experimental.Insights.IngestionUrl,
			SecretName:     cr.Spec.Experimental.Insights.SecretName,
			SecretTokenKey: cr.Spec.Experimental.Insights.SecretTokenKey,
		},
		ControlPlane:    cr.Spec.DataCollection.Metrics.ControlPlane,
		Openshift:       cr.Spec.Openshift,
		Tolerations:     cr.Spec.DataCollection.Tolerations,
		ImagePullSecret: cr.Spec.ImagePullSecret,
	}

	return defaultResources(config)
}

func defaultResources(config ComponentConfig) ComponentConfig {
	if config.NodeCollectorResources.IsEmpty() {
		config.NodeCollectorResources = wf.Resources{
			Requests: wf.Resource{
				CPU:              "200m",
				Memory:           "10Mi",
				EphemeralStorage: "20Mi",
			},
			Limits: wf.Resource{
				CPU:              "1000m",
				Memory:           "256Mi",
				EphemeralStorage: "512Mi",
			},
		}
	}

	if config.ClusterCollectorResources.IsEmpty() {
		config.ClusterCollectorResources = wf.Resources{
			Requests: wf.Resource{
				CPU:              "200m",
				Memory:           "10Mi",
				EphemeralStorage: "20Mi",
			},
			Limits: wf.Resource{
				CPU:              "2000m",
				Memory:           "512Mi",
				EphemeralStorage: "1Gi",
			},
		}
	}
	return config
}
