package patch

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ContainerResources wf.Resources

func (override ContainerResources) Apply(resource *unstructured.Unstructured) {
	containers, _, _ := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "containers")
	if len(containers) == 0 {
		return
	}
	container := containers[0].(map[string]any)
	mergeOverride(override.Limits, "limits", container)
	mergeOverride(override.Requests, "requests", container)
	_ = unstructured.SetNestedSlice(resource.Object, containers, "spec", "template", "spec", "containers")
}

func mergeOverride(override wf.Resource, requirement string, container map[string]any) {
	if len(override.CPU) > 0 {
		_ = unstructured.SetNestedField(container, override.CPU, "resources", requirement, v1.ResourceCPU.String())
	}
	if len(override.Memory) > 0 {
		_ = unstructured.SetNestedField(container, override.Memory, "resources", requirement, v1.ResourceMemory.String())
	}
	if len(override.EphemeralStorage) > 0 {
		_ = unstructured.SetNestedField(container, override.EphemeralStorage, "resources", requirement, v1.ResourceEphemeralStorage.String())
	}
}
