package patch

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Patch func(resource *unstructured.Unstructured)

type ResourcesByName map[string]Patch

func (patches ResourcesByName) Apply(resource *unstructured.Unstructured) {
	if patch, exists := patches[resource.GetName()]; exists {
		patch(resource)
	}
}

type AllResources []Patch

func (patches AllResources) Apply(resource *unstructured.Unstructured) {
	for _, patch := range patches {
		patch(resource)
	}
}
