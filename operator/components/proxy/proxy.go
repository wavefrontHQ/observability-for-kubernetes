package proxy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "proxy"

type ComponentConfig struct {
	// required
	Enable               bool
	ShouldValidate       bool
	ControllerManagerUID string
	Namespace            string
	ClusterName          string
	ClusterUUID          string
	ImageRegistry        string
	WavefrontTokenSecret string
	WavefrontUrl         string
	Resources            wf.Resources
	MetricPort           int
	ProxyVersion         string
	ConfigHash           string
	Replicas             int

	// optional
	Openshift         bool
	SecretHash        string
	ImagePullSecret   string
	LoggingEnable     bool
	DeltaCounterPort  int
	Args              string
	HttpProxy         wf.HttpProxy
	OTLP              wf.OTLP
	Histogram         wf.Histogram
	Tracing           wf.Tracing
	Auth              wf.Auth
	PreprocessorRules wf.PreprocessorRules
}

type Component struct {
	dir    fs.FS
	config ComponentConfig
}

func (component *Component) Name() string {
	return "proxy"
}

func NewComponent(dir fs.FS, componentConfig ComponentConfig) (Component, error) {

	configHashBytes, err := json.Marshal(componentConfig)
	if err != nil {
		return Component{}, errors.New("proxy: problem calculating config hash")
	}
	componentConfig.ConfigHash = components.HashValue(configHashBytes)

	return Component{
		config: componentConfig,
		dir:    dir,
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

	if len(component.config.Namespace) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing namespace", component.Name()))
	}

	if len(component.config.ClusterName) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing cluster name", component.Name()))
	}

	if len(component.config.ClusterUUID) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing cluster uuid", component.Name()))
	}

	if len(component.config.ImageRegistry) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing image registry", component.Name()))
	}

	if len(component.config.WavefrontTokenSecret) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing wavefront token secret", component.Name()))
	}

	if len(component.config.WavefrontUrl) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing wavefront url", component.Name()))
	}

	if result := validation.ValidateResources(&component.config.Resources, util.ProxyName); result.IsError() {
		errs = append(errs, fmt.Errorf("%s: %s", component.Name(), result.Message()))
	}

	if component.config.MetricPort == 0 {
		errs = append(errs, fmt.Errorf("%s: missing metric port", component.Name()))
	}

	if len(component.config.ProxyVersion) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing proxy version", component.Name()))
	}

	if len(component.config.ConfigHash) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing config hash", component.Name()))
	}

	return validation.NewValidationResult(errs)
}

func (component *Component) defaultWorkloadResources() patch.Patch {
	return patch.ByName{
		util.ProxyName: patch.ContainerResources(component.config.Resources),
	}
}

func (component *Component) Resources(builder *components.K8sResourceBuilder) ([]client.Object, []client.Object, error) {
	return builder.Build(component.dir, component.Name(), component.config.Enable, component.config.ControllerManagerUID, component.defaultWorkloadResources(), component.config)
}
