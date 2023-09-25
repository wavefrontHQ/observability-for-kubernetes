package autotracing

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

func FromWavefront(cr *wf.Wavefront) ComponentConfig {
	config := ComponentConfig{
		Enable:               cr.Spec.CanExportData && cr.Spec.Experimental.Autotracing.Enable,
		ControllerManagerUID: cr.Spec.ControllerManagerUID,
		Namespace:            cr.Spec.Namespace,
	}

	return config
}
