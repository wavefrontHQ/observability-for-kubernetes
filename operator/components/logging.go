package components

import (
	"encoding/json"
	"errors"
	"io/fs"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LoggingComponentConfig struct {
	// required
	ClusterName    string
	LoggingVersion string
	ImageRegistry  string
	Namespace      string
	ProxyAddress   string

	// optional
	ProxyAvailableReplicas int
	ImagePullSecret        string
	Tags                   map[string]string
	TagAllowList           map[string][]string
	TagDenyList            map[string][]string
	Tolerations            []wf.Toleration
	Resources              wf.Resources

	// internally set
	ConfigHash           string `json:"-"`
	ControllerManagerUID string `json:"-"`
}

type LoggingComponent struct {
	fs        fs.FS
	DeployDir string
	Config    LoggingComponentConfig
}

func (logging *LoggingComponent) Name() string {
	return "logging"
}

func NewLoggingComponent(componentConfig LoggingComponentConfig, fs fs.FS) (LoggingComponent, error) {

	configHashBytes, err := json.Marshal(componentConfig)
	if err != nil {
		return LoggingComponent{}, errors.New("logging: problem calculating config hash")
	}
	componentConfig.ConfigHash = hashValue(configHashBytes)

	return LoggingComponent{
		Config:    componentConfig,
		fs:        fs,
		DeployDir: DeployDir + "/logging",
	}, nil
}

func (logging *LoggingComponent) Validate() validation.Result {
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
	}

	return validation.Result{}
}

func (logging *LoggingComponent) Resources() ([]client.Object, []client.Object, error) {
	return BuildResources(logging.fs, logging.DeployDir, logging.Name(), logging.Config.ControllerManagerUID, logging.Config)
}
