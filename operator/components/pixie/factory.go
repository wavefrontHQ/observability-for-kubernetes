package pixie

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
)

var HubSources = []string{"socket_tracer", "network_stats", "process_stats"}
var AutoTracingSources = []string{"socket_tracer"}

func FromWavefront(cr *wf.Wavefront) Config {
	config := Config{
		Enable:               cr.Spec.Experimental.Hub.Pixie.Enable || cr.Spec.Experimental.Autotracing.Enable,
		ControllerManagerUID: cr.Spec.ControllerManagerUID,
		ClusterUUID:          cr.Spec.ClusterUUID,
		ClusterName:          cr.Spec.ClusterName,
	}

	if cr.Spec.Experimental.Hub.Pixie.Enable {
		config.StirlingSources = HubSources
		config.PemResources = cr.Spec.Experimental.Hub.Pixie.Pem.Resources
		config.TableStoreLimits = cr.Spec.Experimental.Hub.Pixie.Pem.TableStoreLimits
	} else if cr.Spec.Experimental.Autotracing.Enable {
		config.StirlingSources = AutoTracingSources
		config.PemResources = cr.Spec.Experimental.Autotracing.Pem.Resources
		config.TableStoreLimits = cr.Spec.Experimental.Autotracing.Pem.TableStoreLimits
	}
	return config
}
