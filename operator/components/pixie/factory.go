package pixie

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

func FromWavefront(cr *wf.Wavefront) ComponentConfig {
	config := ComponentConfig{
		Enable:                   false,
		ControllerManagerUID:     cr.Spec.ControllerManagerUID,
		ClusterUUID:              cr.Spec.ClusterUUID,
		ClusterName:              cr.Spec.ClusterName,
		EnableOpAppsOptimization: false,
		PemResources:             wf.Resources{},
	}
	return config
}
