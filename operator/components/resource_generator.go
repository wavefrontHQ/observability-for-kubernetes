package components

import (
	"bytes"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	baseYaml "gopkg.in/yaml.v2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func CreateResources(fs fs.FS, deployDir, componentDir, managerUID string, data any) ([]client.Object, []client.Object, error) {
	files, err := resourceFiles(deployDir, []string{componentDir})
	if err != nil {
		return nil, nil, err
	}

	var resourcesToApply, resourcesToDelete []client.Object
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	for resourceFile, shouldApply := range files {
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
			labels["app.kubernetes.io/component"] = filepath.Base(filepath.Dir(resourceFile))
		}
		resource.SetLabels(labels)

		resource.SetOwnerReferences([]v1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "wavefront-controller-manager",
			UID:        types.UID(managerUID),
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
