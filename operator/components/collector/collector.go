package collector

import (
	"fmt"
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "collector"

type ComponentConfig struct {
	// required
	Enable                    bool
	MetricsEnable             bool
	CustomConfig              string
	ControllerManagerUID      string
	ClusterName               string
	ClusterUUID               string
	EnableDiscovery           bool
	DefaultCollectionInterval string
	ProxyAddress              string
	Namespace                 string
	AvailableReplicas         int
	ImageRegistry             string
	CollectorVersion          string
	ClusterCollectorResources wf.Resources
	NodeCollectorResources    wf.Resources
	CollectorConfigName       string

	// optional
	Filters          wf.Filters
	Tags             map[string]string
	KubernetesEvents KubernetesEvents
	ControlPlane     wf.ControlPlane
	Openshift        bool
	Tolerations      []wf.Toleration
	ImagePullSecret  string
}

type KubernetesEvents struct {
	Enable              bool
	ExternalEndpointURL string
	SecretName          string
}

type Component struct {
	dir    fs.FS
	config ComponentConfig
}

func (collector *Component) Name() string {
	return "collector"
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

func (collector *Component) Resources() ([]client.Object, []client.Object, error) {
	return components.BuildResources(collector.dir, collector.Name(), collector.config.Enable, collector.config.ControllerManagerUID, collector.config)
}
