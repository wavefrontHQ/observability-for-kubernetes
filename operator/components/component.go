package components

import (
	"bytes"
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/util"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	yaml2 "gopkg.in/yaml.v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
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
		if resource.GetKind() == "DaemonSet" || resource.GetKind() == "Deployment" || resource.GetKind() == "StatefulSet" {
			setTemplateComponentLabels(resource, componentName)
		}
		resource.SetOwnerReferences([]v1.OwnerReference{{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
			Name:       "wavefront-controller-manager",
			UID:        types.UID(managerUID),
		}})

		if enabled && resource.GetAnnotations()["wavefront.com/conditionally-provision"] != "false" {
			if componentName == "pixie" {
				scheme := runtime.NewScheme()
				utilruntime.Must(batchv1.AddToScheme(scheme))
				objClient, err := client.New(config.GetConfigOrDie(), client.Options{Scheme: scheme})
				if err != nil {
					return nil, nil, err
				}

				certProvisionerJob, err := getJob(objClient, "cert-provisioner-job", "observability-system")
				if err != nil {
					if apierrors.IsNotFound(err) {
						if resource.GetName() == "cert-provisioner-job" || resource.GetName() == "pl-cert-provisioner-service-account" || resource.GetName() == "pl-cloud-config" || resource.GetName() == "pl-cluster-secrets" {
							resourcesToApply = append(resourcesToApply, resource)
						} else {
							continue
						}
					} else {
						return nil, nil, err
					}
				}

				certProvisioningCompleted := false
				if certProvisionerJob != nil {
					for _, status := range certProvisionerJob.Status.Conditions {
						if status.Type == batchv1.JobComplete && status.Status == corev1.ConditionTrue {
							certProvisioningCompleted = true
						}
					}
				}

				if certProvisioningCompleted {
					resourcesToApply = append(resourcesToApply, resource)
				}
			} else {
				resourcesToApply = append(resourcesToApply, resource)
			}
		} else {
			resourcesToDelete = append(resourcesToDelete, resource)
		}
	}
	return resourcesToApply, resourcesToDelete, nil
}

func setTemplateComponentLabels(resource *unstructured.Unstructured, componentName string) {
	labels, _, _ := unstructured.NestedStringMap(resource.Object, "spec", "template", "metadata", "labels")
	if labels == nil {
		labels = map[string]string{}
	}

	labels["app.kubernetes.io/name"] = "wavefront"
	if labels["app.kubernetes.io/component"] == "" {
		labels["app.kubernetes.io/component"] = componentName
	}

	_ = unstructured.SetNestedStringMap(resource.Object, labels, "spec", "template", "metadata", "labels")
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

func getJob(client client.Client, name, ns string) (*batchv1.Job, error) {
	var job batchv1.Job
	err := client.Get(context.Background(), util.ObjKey(ns, name), &job)
	if err != nil {
		return nil, err
	}

	return &job, nil
}
