package factory

import (
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/autotracing"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/collector"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/logging"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/pixie"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/proxy"
)

func BuildComponents(componentsDir fs.FS, wf *wf.Wavefront) ([]components.Component, error) {
	var created []components.Component

	proxyDir, err := fs.Sub(componentsDir, proxy.DeployDir)
	if err != nil {
		return nil, err
	}

	proxyComponent, err := proxy.NewComponent(proxyDir, proxy.FromWavefront(wf))
	if err != nil {
		return nil, err
	}
	created = append(created, &proxyComponent)

	collectorDir, err := fs.Sub(componentsDir, collector.DeployDir)
	if err != nil {
		return nil, err
	}

	collectorComponent, err := collector.NewComponent(collectorDir, collector.FromWavefront(wf))
	if err != nil {
		return nil, err
	}
	created = append(created, &collectorComponent)

	autotracingDir, err := fs.Sub(componentsDir, autotracing.DeployDir)
	if err != nil {
		return nil, err
	}

	autotracingComponent, err := autotracing.NewComponent(autotracingDir, autotracing.FromWavefront(wf))
	if err != nil {
		return nil, err
	}
	created = append(created, &autotracingComponent)

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

	pixieComponent, _ := pixie.NewComponent(pixieDir, pixie.FromWavefront(wf))
	if err != nil {
		return nil, err
	}
	created = append(created, &pixieComponent)

	return created, err
}
