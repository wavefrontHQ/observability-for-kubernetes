package factory

import (
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/autotracing"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/logging"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/pixie"
)

func BuildComponents(componentsDir fs.FS, wf *wf.Wavefront) ([]components.Component, error) {
	var created []components.Component

	loggingDir, err := fs.Sub(componentsDir, logging.DeployDir)
	if err != nil {
		return nil, err
	}

	loggingComponent, err := logging.NewComponent(loggingDir, logging.FromWavefront(wf))
	if err != nil {
		return nil, err
	}
	created = append(created, &loggingComponent)

	pixieDir, err := fs.Sub(componentsDir, pixie.DeployDir)
	if err != nil {
		return nil, err
	}

	pixieComponent, err := pixie.NewComponent(pixieDir, pixie.FromWavefront(wf))
	if err != nil {
		return nil, err
	}
	created = append(created, &pixieComponent)

	autotracingDir, err := fs.Sub(componentsDir, autotracing.DeployDir)
	if err != nil {
		return nil, err
	}

	autotracingComponent, err := autotracing.NewComponent(autotracingDir, autotracing.FromWavefront(wf))
	if err != nil {
		return nil, err
	}
	created = append(created, &autotracingComponent)

	return created, err
}
