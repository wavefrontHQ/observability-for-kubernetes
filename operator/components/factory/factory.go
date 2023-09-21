package factory

import (
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/logging"
)

func BuildComponents(componentsDir fs.FS, wf *wf.Wavefront) ([]components.Component, error) {
	var created []components.Component
	config := logging.ComponentConfig{
		Enable:          wf.Spec.CanExportData && wf.Spec.DataCollection.Logging.Enable,
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

	loggingDir, err := fs.Sub(componentsDir, logging.DeployDir)
	if err != nil {
		return nil, err
	}

	loggingComponent, err := logging.NewComponent(config, loggingDir)
	if err != nil {
		return nil, err
	}

	created = append(created, &loggingComponent)

	return created, err
}
