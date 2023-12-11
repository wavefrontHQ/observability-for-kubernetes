package pixie

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var HubSources = []string{"socket_tracer", "network_stats", "process_stats"}
var AutoTracingSources = []string{"socket_tracer"}

func FromWavefront(cr *wf.Wavefront, client client.Client) Config {
	config := defaultResources(cr.Spec.ClusterSize, Config{
		Enable:               cr.Spec.Experimental.Hub.Pixie.Enable || cr.Spec.Experimental.Autotracing.Enable,
		ShouldValidate:       cr.Spec.Experimental.Hub.Pixie.Enable || cr.Spec.Experimental.Autotracing.Enable,
		ControllerManagerUID: cr.Spec.ControllerManagerUID,
		ClusterUUID:          cr.Spec.ClusterUUID,
		ClusterName:          cr.Spec.ClusterName,
	})

	if !cr.Spec.Experimental.Pixie.TableStoreLimits.IsEmpty() {
		config.TableStoreLimits = cr.Spec.Experimental.Pixie.TableStoreLimits
	}

	if cr.Spec.Experimental.Hub.Pixie.Enable {
		config.StirlingSources = HubSources
	} else if cr.Spec.Experimental.Autotracing.Enable {
		config.StirlingSources = AutoTracingSources
	}

	if _, err := components.FindSecret(client, util.PixieTLSCertsName, cr.Spec.Namespace); err == nil {
		config.TLSCertsSecretExists = true
	}

	return config
}
