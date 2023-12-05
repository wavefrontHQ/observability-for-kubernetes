package pixie

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components/patch"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/result"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "pixie"

type Config struct {
	Enable               bool
	TLSCertsSecretExists bool
	ControllerManagerUID string
	ClusterUUID          string
	ClusterName          string
	// StirlingSources list of sources to enable on the PEM containers.
	// Specify a source group (kAll, kProd, kMetrics, kTracers, kProfiler, kTCPStats) or individual sources.
	// You can find the names of sources at https://github.com/pixie-io/pixie/blob/release/vizier/v0.14.2/src/stirling/stirling.cc
	StirlingSources             []string
	PEMResources                common.ContainerResources
	TableStoreLimits            wf.TableStoreLimits
	KelvinResources             common.ContainerResources
	QueryBrokerResources        common.ContainerResources
	NATSResources               common.ContainerResources
	MetadataResources           common.ContainerResources
	CertProvisionerJobResources common.ContainerResources
	MaxHTTPBodyBytes            int
}

func (c Config) StirlingSourcesEnv() string {
	return strings.Join(c.StirlingSources, ",")
}

type Component struct {
	dir    fs.FS
	config Config
}

func NewComponent(dir fs.FS, config Config) (Component, error) {
	return Component{
		config: config,
		dir:    dir,
	}, nil
}

func (component *Component) Name() string {
	return "pixie"
}

func (component *Component) Validate() result.Result {
	var errs []error

	if !component.config.Enable {
		return result.Valid
	}

	if len(component.config.ControllerManagerUID) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing controller manager uid", component.Name()))
	}

	if len(component.config.ClusterUUID) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing cluster uuid", component.Name()))
	}

	if len(component.config.ClusterName) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing cluster name", component.Name()))
	}

	if result := validation.ValidateContainerResources(&component.config.PEMResources, util.PixieVizierPEMName); result.IsError() {
		errs = append(errs, fmt.Errorf("%s: %s", component.Name(), result.Message()))
	}

	return result.NewError(errs...)
}

func (component *Component) patch() patch.Patch {
	return patch.ByName{
		util.PixieVizierPEMName:          patch.ContainerResources(component.config.PEMResources),
		util.PixieVizierQueryBrokerName:  patch.ContainerResources(component.config.QueryBrokerResources),
		util.PixieNatsName:               patch.ContainerResources(component.config.NATSResources),
		util.PixieKelvinName:             patch.ContainerResources(component.config.KelvinResources),
		util.PixieVizierMetadataName:     patch.ContainerResources(component.config.MetadataResources),
		util.PixieCertProvisionerJobName: patch.ContainerResources(component.config.CertProvisionerJobResources),
	}
}

func (component *Component) Resources(builder *components.K8sResourceBuilder) ([]client.Object, []client.Object, error) {
	return builder.Build(component.dir, component.Name(), component.config.Enable, component.config.ControllerManagerUID, component.patch(), component.config)
}
