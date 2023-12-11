package logging

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

func FromWavefront(cr *wf.Wavefront) ComponentConfig {
	config := ComponentConfig{
		Enable:          cr.Spec.CanExportData && cr.Spec.DataCollection.Logging.Enable,
		ShouldValidate:  cr.Spec.DataCollection.Logging.Enable,
		ClusterName:     cr.Spec.ClusterName,
		Namespace:       cr.Spec.Namespace,
		LoggingVersion:  cr.Spec.DataCollection.Logging.LoggingVersion,
		ImageRegistry:   cr.Spec.ImageRegistry,
		ImagePullSecret: cr.Spec.ImagePullSecret,

		ProxyAddress:           cr.Spec.DataCollection.Logging.ProxyAddress,
		ProxyAvailableReplicas: cr.Spec.DataExport.WavefrontProxy.AvailableReplicas,
		Tolerations:            cr.Spec.DataCollection.Tolerations,
		Resources:              cr.Spec.DataCollection.Logging.Resources,
		TagAllowList:           cr.Spec.DataCollection.Logging.Filters.TagAllowList,
		TagDenyList:            cr.Spec.DataCollection.Logging.Filters.TagDenyList,
		Tags:                   cr.Spec.DataCollection.Logging.Tags,

		ControllerManagerUID: cr.Spec.ControllerManagerUID,
	}
	return config
}
