package components

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	yaml2 "gopkg.in/yaml.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Component interface {
	Validate() validation.Result
	Resources() (resourcesToApply []client.Object, resourcesToDelete []client.Object, error error)
	Name() string
}

const DeployDir = "components"

func BuildResources(fs fs.FS, componentName string, enabled bool, managerUID string, data any) ([]client.Object, []client.Object, error) {
	files, err := resourceFiles(fs)
	if err != nil {
		return nil, nil, err
	}

	var resourcesToApply, resourcesToDelete []client.Object
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for _, resourceFile := range files {
		templateName := filepath.Base(resourceFile)
		resourceTemplate, err := newTemplate(templateName).ParseFS(fs, resourceFile)
		if err != nil {
			return nil, nil, err
		}
		buffer := bytes.NewBuffer(nil)
		err = resourceTemplate.Execute(buffer, data)
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
			labels["app.kubernetes.io/component"] = componentName
		}
		resource.SetLabels(labels)

		resource.SetOwnerReferences([]v1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "wavefront-controller-manager",
			UID:        types.UID(managerUID),
		}})

		if enabled && resource.GetAnnotations()["wavefront.com/conditionally-provision"] != "false" {
			resourcesToApply = append(resourcesToApply, resource)
		} else {
			resourcesToDelete = append(resourcesToDelete, resource)
		}
	}
	return resourcesToApply, resourcesToDelete, nil
}

func resourceFiles(dir fs.FS) ([]string, error) {
	extension := ".yaml"
	var files []string
	err := fs.WalkDir(dir, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() {
			return nil
		}

		if filepath.Ext(path) == extension {
			files = append(files, path)
		}

		return nil
	})

	return files, err
}

func newTemplate(resourceFile string) *template.Template {
	fMap := template.FuncMap{
		"toYaml": func(v interface{}) string {
			data, err := yaml2.Marshal(v)
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
