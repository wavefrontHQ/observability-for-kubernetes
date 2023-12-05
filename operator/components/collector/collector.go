package collector

import (
	"fmt"
	"io/fs"

	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/result"
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
	ClusterCollectorResources common.ContainerResources
	NodeCollectorResources    common.ContainerResources
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

func (component *Component) Name() string {
	return "collector"
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

	if result := validation.ValidateContainerResources(&component.config.ClusterCollectorResources, util.ClusterCollectorName); result.IsError() {
		errs = append(errs, fmt.Errorf("%s: %s", component.Name(), result.Message()))
	}

	if component.config.MetricsEnable {
		if result := validation.ValidateContainerResources(&component.config.NodeCollectorResources, util.NodeCollectorName); result.IsError() {
			errs = append(errs, fmt.Errorf("%s: %s", component.Name(), result.Message()))
		}
	}

	if component.config.KubernetesEvents.Enable {
		if len(component.config.KubernetesEvents.IngestionUrl) == 0 {
			errs = append(errs, fmt.Errorf("%s: missing insights ingestion url", component.Name()))
		}
	}

	return result.NewError(errs...)
}

func (component *Component) Resources(builder *components.K8sResourceBuilder) ([]client.Object, []client.Object, error) {
	return builder.Build(component.dir, component.Name(), component.config.Enable, component.config.ControllerManagerUID, component.defaultWorkloadResources(), component.config)
}

func (component *Component) defaultWorkloadResources() patch.Patch {
	return patch.ByName{
		util.NodeCollectorName:    patch.ContainerResources(component.config.NodeCollectorResources),
		util.ClusterCollectorName: patch.ContainerResources(component.config.ClusterCollectorResources),
	}
}
