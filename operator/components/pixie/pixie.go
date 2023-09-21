package pixie

import (
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "pixie"

type ComponentConfig struct {
	// common
	Enable               bool
	ControllerManagerUID string

	// required
	ClusterUUID              string
	ClusterName              string
	EnableOpAppsOptimization bool
	PemResources			 wf.Resources
}

type Component struct {
	dir    fs.FS
	config ComponentConfig
}

func (pixie *Component) Name() string {
	return "pixie"
}

func NewComponent(componentConfig ComponentConfig, dir fs.FS) (Component, error) {

	return Component{
		config: componentConfig,
		dir:    dir,
	}, nil
}

func (pixie *Component) Validate() validation.Result {
	return validation.Result{}
}

func (pixie *Component) Resources() ([]client.Object, []client.Object, error) {
	return components.BuildResources(pixie.dir, pixie.Name(), pixie.config.Enable, pixie.config.ControllerManagerUID, pixie.config)
}
