package components

import (
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
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

	// calculated
	ConfigHash string `json:"-"`
}

type LoggingComponent struct {
	Config LoggingComponentConfig
}

func (logging *LoggingComponent) PreprocessAndValidate() validation.Result {
	if len(logging.Config.ClusterName) == 0 {
		return validation.NewErrorResult(errors.New("missing cluster name"))
	}

	if len(logging.Config.Namespace) == 0 {
		return validation.NewErrorResult(errors.New("missing namespace"))
	}

	if len(logging.Config.LoggingVersion) == 0 {
		return validation.NewErrorResult(errors.New("missing logging version"))
	}

	if len(logging.Config.ImageRegistry) == 0 {
		return validation.NewErrorResult(errors.New("missing image registry"))
	}

	if len(logging.Config.ProxyAddress) == 0 {
		return validation.NewErrorResult(errors.New("missing proxy address"))
	}

	configHashBytes, err := json.Marshal(logging.Config)
	if err != nil {
		return validation.NewErrorResult(errors.New("problem calculating config hash"))
	}
	logging.Config.ConfigHash = hashValue(configHashBytes)

	return validation.Result{}
}

func (logging *LoggingComponent) ReadAndCreateResources() error {
	return nil
}

func (logging *LoggingComponent) ReadAndDeleteResources() error {
	return nil
}

func NewLoggingComponent(componentConfig LoggingComponentConfig) LoggingComponent {
	return LoggingComponent{
		Config: componentConfig,
	}
}

func hashValue(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)

	return fmt.Sprintf("%x", h.Sum(nil))
}
