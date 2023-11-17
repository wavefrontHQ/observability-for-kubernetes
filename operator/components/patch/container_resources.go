package patch

import (
	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

type ContainerResources struct {
	Requests ContainerResource
	Limits   ContainerResource
}

type ContainerResource struct {
	CPU              string
	Memory           string
	EphemeralStorage string
}

func FromWFResources(resources wf.Resources) ContainerResources {
	return ContainerResources{
		Requests: fromWFResource(resources.Requests),
		Limits:   fromWFResource(resources.Limits),
	}
}

func fromWFResource(resource wf.Resource) ContainerResource {
	return ContainerResource{
		CPU:              resource.CPU,
		Memory:           resource.Memory,
		EphemeralStorage: resource.EphemeralStorage,
	}
}

func FromRCResources(resources rc.Resources) ContainerResources {
	return ContainerResources{
		Requests: fromRCResource(resources.Requests),
		Limits:   fromRCResource(resources.Limits),
	}
}

func fromRCResource(resource rc.Resource) ContainerResource {
	return ContainerResource{
		CPU:              resource.CPU,
		Memory:           resource.Memory,
		EphemeralStorage: resource.EphemeralStorage,
	}
}

func (r ContainerResource) Empty() bool {
	return len(r.CPU) == 0 && len(r.Memory) == 0 && len(r.EphemeralStorage) == 0
}

func (r ContainerResources) Apply(resource *unstructured.Unstructured) {
	containers, _, _ := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "containers")
	if len(containers) == 0 {
		return
	}
	container := containers[0].(map[string]any)
	applyContainerResource(r.Limits, "limits", container)
	if !r.Requests.Empty() {
		applyContainerResource(r.Requests, "requests", container)
	} else {
		applyContainerResource(r.Limits, "requests", container)
	}
	_ = unstructured.SetNestedSlice(resource.Object, containers, "spec", "template", "spec", "containers")
}

func applyContainerResource(override ContainerResource, requirement string, container map[string]any) {
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
