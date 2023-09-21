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
	fs     fs.FS
	Config ComponentConfig
}

func (logging *Component) Name() string {
	return "logging"
}

func NewComponent(componentConfig ComponentConfig, fs fs.FS) (Component, error) {

	configHashBytes, err := json.Marshal(componentConfig)
	if err != nil {
		return Component{}, errors.New("logging: problem calculating config hash")
	}
	componentConfig.ConfigHash = components.HashValue(configHashBytes)

	return Component{
		Config: componentConfig,
		fs:     fs,
	}, nil
}

func (logging *Component) Validate() validation.Result {
	if !logging.Config.Enable {
		return validation.Result{}
	}
	if len(logging.Config.ClusterName) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing cluster name"))
	}

	if len(logging.Config.Namespace) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing namespace"))
	}

	if len(logging.Config.LoggingVersion) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing log image version"))
	}

	if len(logging.Config.ImageRegistry) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing image registry"))
	}

	if len(logging.Config.ProxyAddress) == 0 {
		return validation.NewErrorResult(errors.New("logging: missing proxy address"))
	} else if !strings.HasPrefix(logging.Config.ProxyAddress, "http") {
		return validation.NewErrorResult(errors.New(fmt.Sprintf("logging: proxy address (%s) must start with http", logging.Config.ProxyAddress)))
	}

	return validation.Result{}
}

func (logging *Component) Resources() ([]client.Object, []client.Object, error) {
	return components.BuildResources(logging.fs, logging.Name(), logging.Config.Enable, logging.Config.ControllerManagerUID, logging.Config)
}
