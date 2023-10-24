package pixie

import (
	"fmt"
	"io/fs"
	"strings"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "pixie"

type Config struct {
	Enable               bool
	ControllerManagerUID string
	ClusterUUID          string
	ClusterName          string
	// StirlingSources list of sources to enable on the PEM containers.
	// Specify a source group (kAll, kProd, kMetrics, kTracers, kProfiler, kTCPStats) or individual sources.
	// You can find the names of sources at https://github.com/pixie-io/pixie/blob/release/vizier/v0.14.2/src/stirling/stirling.cc
	StirlingSources      []string
	PEMResources         wf.Resources
	TableStoreLimits     wf.TableStoreLimits
	KelvinResources      wf.Resources
	QueryBrokerResources wf.Resources
	NATSResources        wf.Resources
	MetadataResources    wf.Resources
	MaxHTTPBodyBytes     int
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

func (pc *Component) Name() string {
	return "pixie"
}

func (pc *Component) Validate() validation.Result {
	var errs []error

	if !pc.config.Enable {
		return validation.Result{}
	}

	if len(pc.config.ControllerManagerUID) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing controller manager uid", pc.Name()))
	}

	if len(pc.config.ClusterUUID) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing cluster uuid", pc.Name()))
	}

	if len(pc.config.ClusterName) == 0 {
		errs = append(errs, fmt.Errorf("%s: missing cluster name", pc.Name()))
	}

	if result := validation.ValidateResources(&pc.config.PEMResources, util.PixieVizierPEMName); result.IsError() {
		errs = append(errs, fmt.Errorf("%s: %s", pc.Name(), result.Message()))
	}

	return validation.NewValidationResult(errs)
}

func (pc *Component) resourceOverrides() map[string]wf.Resources {
	return map[string]wf.Resources{
		util.PixieVizierPEMName:         pc.config.PEMResources,
		util.PixieVizierQueryBrokerName: pc.config.QueryBrokerResources,
		util.PixieNatsName:              pc.config.NATSResources,
		util.PixieKelvinName:            pc.config.KelvinResources,
		util.PixieVizierMetadataName:    pc.config.MetadataResources,
	}
}

func (pc *Component) Resources() ([]client.Object, []client.Object, error) {
	return components.BuildResources(pc.dir, pc.Name(), pc.config.Enable, pc.config.ControllerManagerUID, pc.resourceOverrides(), pc.config)
}
