package autotracing

import (
	"fmt"
	"io/fs"

	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/result"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "autotracing"

type ComponentConfig struct {
	// required
	Enable               bool
	ControllerManagerUID string
	Namespace            string
}

type Component struct {
	dir    fs.FS
	config ComponentConfig
}

func (autotracing *Component) Name() string {
	return "autotracing"
}

func NewComponent(dir fs.FS, componentConfig ComponentConfig) (Component, error) {

	return Component{
		config: componentConfig,
		dir:    dir,
	}, nil
}

func (component *Component) Validate() result.Result {
	var errs []error

	if !component.config.Enable {
		return result.Valid
	}

	if len(component.config.ControllerManagerUID) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing controller manager uid", component.Name()))
	}

	if len(component.config.Namespace) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing namespace", component.Name()))
	}

	return result.NewError(errs...)
}

func (autotracing *Component) Resources(builder *components.K8sResourceBuilder) ([]client.Object, []client.Object, error) {
	return builder.Build(autotracing.dir, autotracing.Name(), autotracing.config.Enable, autotracing.config.ControllerManagerUID, nil, autotracing.config)
}
