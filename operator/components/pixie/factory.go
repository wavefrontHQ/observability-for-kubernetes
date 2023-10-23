package pixie

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

var HubSources = []string{"socket_tracer", "network_stats", "process_stats"}
var AutoTracingSources = []string{"socket_tracer"}

func FromWavefront(cr *wf.Wavefront) Config {
	config := defaultResources(cr.Spec.ClusterSize, Config{
		Enable:               cr.Spec.Experimental.Hub.Pixie.Enable || cr.Spec.Experimental.Autotracing.Enable,
		ControllerManagerUID: cr.Spec.ControllerManagerUID,
		ClusterUUID:          cr.Spec.ClusterUUID,
		ClusterName:          cr.Spec.ClusterName,
	})

	if cr.Spec.Experimental.Hub.Pixie.Enable {
		config.StirlingSources = HubSources
	} else if cr.Spec.Experimental.Autotracing.Enable {
		config.StirlingSources = AutoTracingSources
	}
	return config
}
