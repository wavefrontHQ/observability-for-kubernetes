package proxy

import wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"

func FromWavefront(cr *wf.Wavefront) ComponentConfig {
	config := ComponentConfig{
		// required
		Enable:               cr.Spec.DataExport.WavefrontProxy.Enable,
		ShouldValidate:       cr.Spec.DataExport.WavefrontProxy.Enable,
		ControllerManagerUID: cr.Spec.ControllerManagerUID,
		Namespace:            cr.Spec.Namespace,
		ClusterName:          cr.Spec.ClusterName,
		ClusterUUID:          cr.Spec.ClusterUUID,
		ImageRegistry:        cr.Spec.ImageRegistry,
		WavefrontTokenSecret: cr.Spec.WavefrontTokenSecret,
		WavefrontUrl:         cr.Spec.WavefrontUrl,
		Resources:            cr.Spec.DataExport.WavefrontProxy.Resources,
		MetricPort:           cr.Spec.DataExport.WavefrontProxy.MetricPort,
		ProxyVersion:         cr.Spec.DataExport.WavefrontProxy.ProxyVersion,
		ConfigHash:           cr.Spec.DataExport.WavefrontProxy.ConfigHash,
		SecretHash:           cr.Spec.DataExport.WavefrontProxy.SecretHash,
		Replicas:             cr.Spec.DataExport.WavefrontProxy.Replicas,

		// optional
		Openshift:         cr.Spec.Openshift,
		ImagePullSecret:   cr.Spec.ImagePullSecret,
		LoggingEnable:     cr.Spec.DataCollection.Logging.Enable,
		DeltaCounterPort:  cr.Spec.DataExport.WavefrontProxy.DeltaCounterPort,
		Args:              cr.Spec.DataExport.WavefrontProxy.Args,
		HttpProxy:         cr.Spec.DataExport.WavefrontProxy.HttpProxy,
		OTLP:              cr.Spec.DataExport.WavefrontProxy.OTLP,
		Histogram:         cr.Spec.DataExport.WavefrontProxy.Histogram,
		Tracing:           cr.Spec.DataExport.WavefrontProxy.Tracing,
		Auth:              cr.Spec.DataExport.WavefrontProxy.Auth,
		PreprocessorRules: cr.Spec.DataExport.WavefrontProxy.PreprocessorRules,
	}

	return config
}
