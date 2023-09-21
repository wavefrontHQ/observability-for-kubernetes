package factory

import (
	"os"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/logging"
)

func BuildComponents(wf *wf.Wavefront) (map[components.Component]bool, error) {
	createdComponents := make(map[components.Component]bool)
	config := logging.ComponentConfig{
		ClusterName:     wf.Spec.ClusterName,
		Namespace:       wf.Spec.Namespace,
		LoggingVersion:  wf.Spec.DataCollection.Logging.LoggingVersion,
		ImageRegistry:   wf.Spec.ImageRegistry,
		ImagePullSecret: wf.Spec.ImagePullSecret,

		ProxyAddress:           wf.Spec.DataCollection.Logging.ProxyAddress,
		ProxyAvailableReplicas: wf.Spec.DataExport.WavefrontProxy.AvailableReplicas,
		Tolerations:            wf.Spec.DataCollection.Tolerations,
		Resources:              wf.Spec.DataCollection.Logging.Resources,
		TagAllowList:           wf.Spec.DataCollection.Logging.Filters.TagAllowList,
		TagDenyList:            wf.Spec.DataCollection.Logging.Filters.TagDenyList,
		Tags:                   wf.Spec.DataCollection.Logging.Tags,

		ControllerManagerUID: wf.Spec.ControllerManagerUID,
	}

	loggingComponent, err := logging.NewComponent(config, os.DirFS(components.DeployDir))
	if err != nil {
		return nil, err
	}

	createdComponents[&loggingComponent] = wf.Spec.CanExportData && wf.Spec.DataCollection.Logging.Enable
	return createdComponents, err
}
