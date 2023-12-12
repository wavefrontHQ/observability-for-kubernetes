package logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "logging"

type ComponentConfig struct {
	// required
	Enable               bool
	ShouldValidate       bool
	ClusterName          string
	LoggingVersion       string
	ImageRegistry        string
	Namespace            string
	ProxyAddress         string
	ControllerManagerUID string

	// optional
	ProxyAvailableReplicas int
	ImagePullSecret        string
	Tags                   map[string]string
	TagAllowList           map[string][]string
	TagDenyList            map[string][]string
	Tolerations            []wf.Toleration
	Resources              wf.Resources

	// internal use only
	ConfigHash string
}

type Component struct {
	dir    fs.FS
	config ComponentConfig
}

func (logging *Component) Name() string {
	return "logging"
}

func NewComponent(fs fs.FS, componentConfig ComponentConfig) (Component, error) {

	configHashBytes, err := json.Marshal(componentConfig)
	if err != nil {
		return Component{}, errors.New("logging: problem calculating config hash")
	}
	componentConfig.ConfigHash = components.HashValue(configHashBytes)

	return Component{
		config: componentConfig,
		dir:    fs,
	}, nil
}

func (component *Component) Validate() validation.Result {
	var errs []error

	if !component.config.ShouldValidate {
		return validation.Result{}
	}
	if len(component.config.ControllerManagerUID) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing controller manager uid", component.Name()))
	}

	if len(component.config.ClusterName) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing cluster name", component.Name()))
	}

	if len(component.config.Namespace) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing namespace", component.Name()))
	}

	if len(component.config.LoggingVersion) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing log image version", component.Name()))
	}

	if len(component.config.ImageRegistry) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing image registry", component.Name()))
	}

	if len(component.config.ProxyAddress) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing proxy address", component.Name()))
	} else if !strings.HasPrefix(component.config.ProxyAddress, "http") {
		errs = append(errs, fmt.Errorf("logging: proxy address (%s) must start with http", component.config.ProxyAddress))
	}

	if result := validation.ValidateResources(&component.config.Resources, util.LoggingName); result.IsError() {
		errs = append(errs, fmt.Errorf("%s: %s", component.Name(), result.Message()))
	}

	return validation.NewValidationResult(errs)
}

func (logging *Component) defaultWorkloadResources() patch.Patch {
	return patch.ByName{
		util.LoggingName: patch.ContainerResources(logging.config.Resources),
	}
}

func (logging *Component) Resources(builder *components.K8sResourceBuilder) ([]client.Object, []client.Object, error) {
	return builder.Build(logging.dir, logging.Name(), logging.config.Enable, logging.config.ControllerManagerUID, logging.defaultWorkloadResources(), logging.config)
}
