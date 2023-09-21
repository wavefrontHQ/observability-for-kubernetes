package factory

import (
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/logging"
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

	return created, err
}
