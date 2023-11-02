package collector

import (
	"fmt"
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "collector"

type ComponentConfig struct {
	// required
	Enable                    bool
	MetricsEnable             bool
	ControllerManagerUID      string
	ClusterName               string
	ClusterUUID               string
	DefaultCollectionInterval string
	ProxyAddress              string
	Namespace                 string
	ProxyAvailableReplicas    int
	ImageRegistry             string
	CollectorVersion          string
	ClusterCollectorResources wf.Resources
	NodeCollectorResources    wf.Resources
	CollectorConfigName       string

	// optional
	CustomConfig     string
	Filters          wf.Filters
	EnableDiscovery  bool
	Tags             map[string]string
	KubernetesEvents KubernetesEvents
	ControlPlane     wf.ControlPlane
	Openshift        bool
	Tolerations      []wf.Toleration
	ImagePullSecret  string
}

type KubernetesEvents struct {
	Enable         bool
	IngestionUrl   string
	SecretName     string
	SecretTokenKey string
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

	if result := validation.ValidateResources(&component.config.ClusterCollectorResources, util.ClusterCollectorName); result.IsError() {
		errs = append(errs, fmt.Errorf("%s: %s", component.Name(), result.Message()))
	}

	if component.config.MetricsEnable {
		if result := validation.ValidateResources(&component.config.NodeCollectorResources, util.NodeCollectorName); result.IsError() {
			errs = append(errs, fmt.Errorf("%s: %s", component.Name(), result.Message()))
		}
	}

	if component.config.KubernetesEvents.Enable {
		if len(component.config.KubernetesEvents.IngestionUrl) == 0 {
			errs = append(errs, fmt.Errorf("%s: missing insights ingestion url", component.Name()))
		}
	}

	return validation.NewValidationResult(errs)
}

func (collector *Component) Resources(builder *components.K8sResourceBuilder) ([]client.Object, []client.Object, error) {
	return builder.Build(collector.dir, collector.Name(), collector.config.Enable, collector.config.ControllerManagerUID, collector.defaultWorkloadResources(), collector.config)
}

func (collector *Component) defaultWorkloadResources() map[string]wf.Resources {
	return map[string]wf.Resources{
		util.NodeCollectorName:    collector.config.NodeCollectorResources,
		util.ClusterCollectorName: collector.config.ClusterCollectorResources,
	}
}
