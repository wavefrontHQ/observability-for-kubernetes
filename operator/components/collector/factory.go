package collector

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

func FromWavefront(cr *wf.Wavefront) ComponentConfig {
	config := ComponentConfig{
		// required
		Enable:                    (cr.Spec.CanExportData && cr.Spec.DataCollection.Metrics.Enable || cr.Spec.Experimental.Insights.Enable),
		MetricsEnable:             cr.Spec.CanExportData && cr.Spec.DataCollection.Metrics.Enable,
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
			Enable:              cr.Spec.Experimental.Insights.Enable,
			ExternalEndpointURL: cr.Spec.Experimental.Insights.ExternalEndpointURL,
			SecretName:          cr.Spec.Experimental.Insights.SecretName,
		},
		ControlPlane:    cr.Spec.DataCollection.Metrics.ControlPlane,
		Openshift:       cr.Spec.Openshift,
		Tolerations:     cr.Spec.DataCollection.Tolerations,
		ImagePullSecret: cr.Spec.ImagePullSecret,
	}

	return config
}
