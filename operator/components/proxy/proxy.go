package proxy

import (
	"fmt"
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "proxy"

type ComponentConfig struct {
	// required
	ControllerManagerUID string
	Namespace            string
	ClusterName          string
	ClusterUUID          string
	ImageRegistry        string
	WavefrontTokenSecret string
	WavefrontUrl         string
	Enable               bool
	Resources            wf.Resources
	MetricPort           int
	ProxyVersion         string
	ConfigHash           string
	SecretHash           string
	Replicas             int

	// optional
	Openshift         bool
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

func (proxy *Component) Name() string {
	return "proxy"
}

func NewComponent(dir fs.FS, componentConfig ComponentConfig) (Component, error) {

	return Component{
		config: componentConfig,
		dir:    dir,
	}, nil
}

func (component *Component) Validate() validation.Result {
	var errs []error

	if !component.config.Enable {
		return validation.Result{}
	}

	if len(component.config.ControllerManagerUID) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing controller manager uid", component.Name()))
	}

	if len(component.config.Namespace) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing namespace", component.Name()))
	}

	return validation.NewValidationResult(errs)
}

func (proxy *Component) Resources() ([]client.Object, []client.Object, error) {
	return components.BuildResources(proxy.dir, proxy.Name(), proxy.config.Enable, proxy.config.ControllerManagerUID, proxy.config)
}
