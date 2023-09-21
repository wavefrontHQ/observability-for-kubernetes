package logging

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/components"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const DeployDir = "logging"

type ComponentConfig struct {
	// required
	Enable               bool
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

func (logging *Component) Validate() validation.Result {
	if !logging.config.Enable {
		return validation.Result{}
	}
	if len(logging.config.ClusterName) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing cluster name"))
	}

	if len(logging.config.Namespace) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing namespace"))
	}

	if len(logging.config.LoggingVersion) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing log image version"))
	}

	if len(logging.config.ImageRegistry) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing image registry"))
	}

	if len(logging.config.ProxyAddress) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing proxy address"))
	} else if !strings.HasPrefix(logging.config.ProxyAddress, "http") {
		return validation.NewErrorResult(fmt.Errorf("logging: proxy address (%s) must start with http", logging.config.ProxyAddress))
	}

	return validation.Result{}
}

func (logging *Component) Resources() ([]client.Object, []client.Object, error) {
	return components.BuildResources(logging.dir, logging.Name(), logging.config.Enable, logging.config.ControllerManagerUID, logging.config)
}
