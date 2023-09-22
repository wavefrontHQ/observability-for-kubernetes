package pixie

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

func FromWavefront(cr *wf.Wavefront) ComponentConfig {
	config := ComponentConfig{
		Enable:                   cr.Spec.Experimental.Hub.Pixie.Enable || cr.Spec.Experimental.Autotracing.Enable,
		ControllerManagerUID:     cr.Spec.ControllerManagerUID,
		ClusterUUID:              cr.Spec.ClusterUUID,
		ClusterName:              cr.Spec.ClusterName,
		EnableOpAppsOptimization: false,
		PemResources:             wf.Resources{},
	}

	if cr.Spec.Experimental.Hub.Pixie.Enable {
		config.PemResources = cr.Spec.Experimental.Hub.Pixie.Pem.Resources
	} else if cr.Spec.Experimental.Autotracing.Enable {
		config.EnableOpAppsOptimization = true
		config.PemResources = cr.Spec.Experimental.Autotracing.Pem.Resources
	}
	return config
}
