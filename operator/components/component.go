package components

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	yaml2 "gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Component interface {
	Validate() validation.Result
	Resources(builder *K8sResourceBuilder) (resourcesToApply []client.Object, resourcesToDelete []client.Object, error error)
	Name() string
}

const DeployDir = "components"

type K8sResourceBuilder struct {
	containerResourceOverrides map[string]wf.Resources
}

func NewK8sResourceBuilder(containerResourceOverrides map[string]wf.Resources) *K8sResourceBuilder {
	return &K8sResourceBuilder{containerResourceOverrides: containerResourceOverrides}
}

func (rb *K8sResourceBuilder) Build(fs fs.FS, componentName string, enabled bool, managerUID string, containerResourceDefaults map[string]wf.Resources, data any) ([]client.Object, []client.Object, error) {
	files, err := resourceFiles(fs)
	if err != nil {
		return nil, nil, err
	}

	var resourcesToApply, resourcesToDelete []client.Object
	var resourceDecoder = yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	mergedContainerResourceOverrides := mergeResources(containerResourceDefaults, rb.containerResourceOverrides)
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

		setResourceComponentLabels(resource, componentName)
		_, hasTemplate, _ := unstructured.NestedMap(resource.Object, "spec", "template")
		if hasTemplate {
			setTemplateComponentLabels(resource, componentName)
			if err := setTemplateResources(resource, mergedContainerResourceOverrides); err != nil {
				return nil, nil, err
			}
		}
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

func mergeResources(defaults map[string]wf.Resources, overrides map[string]wf.Resources) map[string]wf.Resources {
	merged := map[string]wf.Resources{}
	for name, resources := range defaults {
		merged[name] = resources
	}
	for name, resources := range overrides {
		defaultResources := merged[name]
		defaultResources.Requests = mergeResource(defaultResources.Requests, resources.Requests)
		defaultResources.Limits = mergeResource(defaultResources.Limits, resources.Limits)
		merged[name] = defaultResources
	}
	return merged
}

func mergeResource(a, b wf.Resource) wf.Resource {
	if b.CPU != "" {
		a.CPU = b.CPU
	}
	if b.Memory != "" {
		a.Memory = b.Memory
	}
	if b.EphemeralStorage != "" {
		a.EphemeralStorage = b.EphemeralStorage
	}
	return a
}

func setTemplateResources(workload *unstructured.Unstructured, workloadResources map[string]wf.Resources) error {
	override, exists := workloadResources[workload.GetName()]
	if !exists {
		return fmt.Errorf("workload resource not specified for %s", workload.GetName())
	}

	r := map[string]any{
		"limits": map[string]any{
			corev1.ResourceCPU.String():    override.Limits.CPU,
			corev1.ResourceMemory.String(): override.Limits.Memory,
		},
		"requests": map[string]any{
			corev1.ResourceCPU.String():    override.Requests.CPU,
			corev1.ResourceMemory.String(): override.Requests.Memory,
		},
	}

	if override.Limits.EphemeralStorage != "" {
		r["limits"].(map[string]any)[corev1.ResourceEphemeralStorage.String()] = override.Limits.EphemeralStorage
	}
	if override.Requests.EphemeralStorage != "" {
		r["requests"].(map[string]any)[corev1.ResourceEphemeralStorage.String()] = override.Requests.EphemeralStorage
	}

	containers, _, _ := unstructured.NestedSlice(workload.Object, "spec", "template", "spec", "containers")
	container := containers[0].(map[string]any)
	container["resources"] = r
	_ = unstructured.SetNestedSlice(workload.Object, containers, "spec", "template", "spec", "containers")
	return nil
}

func setResourceComponentLabels(resource *unstructured.Unstructured, componentName string) {
	labels := resource.GetLabels()
	labels = updateComponentLabels(componentName, labels)
	resource.SetLabels(labels)
}

func setTemplateComponentLabels(resource *unstructured.Unstructured, componentName string) {
	labels, _, _ := unstructured.NestedStringMap(resource.Object, "spec", "template", "metadata", "labels")
	labels = updateComponentLabels(componentName, labels)
	_ = unstructured.SetNestedStringMap(resource.Object, labels, "spec", "template", "metadata", "labels")
}

func updateComponentLabels(componentName string, labels map[string]string) map[string]string {
	if labels == nil {
		labels = map[string]string{}
	}
	labels["app.kubernetes.io/name"] = "wavefront"
	if labels["app.kubernetes.io/component"] == "" {
		labels["app.kubernetes.io/component"] = componentName
	}
	return labels
}

func resourceFiles(dir fs.FS) ([]string, error) {
	extension := ".yaml"
	var files []string
	err := fs.WalkDir(dir, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if entry.IsDir() && path != "." {
			return fs.SkipDir
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
