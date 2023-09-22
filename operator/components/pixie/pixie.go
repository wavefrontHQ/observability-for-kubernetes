package pixie

import (
	"fmt"
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "pixie"

type ComponentConfig struct {
	// required
	Enable               bool
	ControllerManagerUID string
	ClusterUUID          string
	ClusterName          string

	// optional
	EnableOpAppsOptimization bool
	PemResources             wf.Resources
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

func (component *Component) Validate() validation.Result {
	if !component.config.Enable {
		return validation.Result{}
	}

	if len(component.config.ControllerManagerUID) == 0 {
		return validation.NewErrorResult(fmt.Errorf("%s: missing controller manager uid", component.Name()))
	}

	if len(component.config.ClusterUUID) == 0 {
		return validation.NewErrorResult(fmt.Errorf("%s: missing cluster uuid", component.Name()))
	}

	if len(component.config.ClusterName) == 0 {
		return validation.NewErrorResult(fmt.Errorf("%s: missing cluster name", component.Name()))
	}

	return validation.Result{}
}

func (pixie *Component) Resources() ([]client.Object, []client.Object, error) {
	return components.BuildResources(pixie.dir, pixie.Name(), pixie.config.Enable, pixie.config.ControllerManagerUID, pixie.config)
}
