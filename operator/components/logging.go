package components

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	baseYaml "gopkg.in/yaml.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
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
	ConfigHash           string `json:"-"`
	ControllerManagerUID string `json:"-"`
}

type LoggingComponent struct {
	fs        fs.FS
	DeployDir string
	Config    LoggingComponentConfig
}

func (logging *LoggingComponent) TemplateDirectory() string {
	return "logging"
}

func (logging *LoggingComponent) Name() string {
	return "logging"
}

func (logging *LoggingComponent) PreprocessAndValidate() validation.Result {
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

	configHashBytes, err := json.Marshal(logging.Config)
	if err != nil {
		return validation.NewErrorResult(errors.New("logging: problem calculating config hash"))
	}
	logging.Config.ConfigHash = hashValue(configHashBytes)

	return validation.Result{}
}

func NewLoggingComponent(componentConfig LoggingComponentConfig, fs fs.FS) LoggingComponent {
	return LoggingComponent{
		Config:    componentConfig,
		fs:        fs,
		DeployDir: DeployDir + "/logging",
	}
}

func hashValue(bytes []byte) string {
	h := sha1.New()
	h.Write(bytes)

	return fmt.Sprintf("%x", h.Sum(nil))
}

func (logging *LoggingComponent) Resources() ([]client.Object, []client.Object, error) {
	// TODO Move Resources functionality of this function to a ComponentResourceGenerator passing the appropriate values
	files, err := resourceFiles(logging.DeployDir, []string{logging.TemplateDirectory()})
	if err != nil {
		return nil, nil, err
	}

	var resourcesToApply, resourcesToDelete []client.Object
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for resourceFile, shouldApply := range files {
		templateName := filepath.Base(resourceFile)
		resourceTemplate, err := newTemplate(templateName).ParseFS(logging.fs, resourceFile)
		if err != nil {
			return nil, nil, err
		}
		buffer := bytes.NewBuffer(nil)
		err = resourceTemplate.Execute(buffer, logging.Config)
		if err != nil {
			return nil, nil, err
		}

		resourceYAML := buffer.String()
		resource := &unstructured.Unstructured{}
		_, _, err = resourceDecoder.Decode([]byte(resourceYAML), nil, resource)
		if err != nil {
			return nil, nil, err
		}

		labels := resource.GetLabels()
		if labels == nil {
			labels = map[string]string{}
		}
		labels["app.kubernetes.io/name"] = "wavefront"
		if labels["app.kubernetes.io/component"] == "" {
			labels["app.kubernetes.io/component"] = filepath.Base(filepath.Dir(resourceFile))
		}
		resource.SetLabels(labels)

		resource.SetOwnerReferences([]v1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "wavefront-controller-manager",
			UID:        types.UID(logging.Config.ControllerManagerUID),
		}})

		if shouldApply && resource.GetAnnotations()["wavefront.com/conditionally-provision"] != "false" {
			resourcesToApply = append(resourcesToApply, resource)
		} else {
			resourcesToDelete = append(resourcesToDelete, resource)
		}
	}
	return resourcesToApply, resourcesToDelete, nil
}

func resourceFiles(deployDir string, dirsToApply []string) (map[string]bool, error) {
	files := make(map[string]bool)
	suffix := "yaml"
	var currentDir string
	err := filepath.WalkDir(deployDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			currentDir = entry.Name()
		}

		if strings.HasSuffix(path, suffix) {
			filePath := strings.Replace(path, deployDir+"/", "logging/", 1)
			if contains(dirsToApply, currentDir) {
				files[filePath] = true
			} else {
				files[filePath] = false
			}
		}

		return nil
	})

	return files, err
}

func newTemplate(resourceFile string) *template.Template {
	fMap := template.FuncMap{
		"toYaml": func(v interface{}) string {
			data, err := baseYaml.Marshal(v)
			if err != nil {
				log.Log.Error(err, "error in toYaml")
				return ""
			}
			return strings.TrimSuffix(string(data), "\n")
		},
		"indent": func(spaces int, v string) string {
			pad := strings.Repeat(" ", spaces)
			return pad + strings.Replace(v, "\n", "\n"+pad, -1)
		},
	}

	return template.New(resourceFile).Funcs(fMap)
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
