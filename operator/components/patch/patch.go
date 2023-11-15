package patch

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Patch interface {
	Apply(resource *unstructured.Unstructured)
}

type ApplyFn func(resource *unstructured.Unstructured)

func (a ApplyFn) Apply(resource *unstructured.Unstructured) {
	a(resource)
}

type ByName map[string]Patch

func (patches ByName) Apply(resource *unstructured.Unstructured) {
	if patch, exists := patches[resource.GetName()]; exists {
		patch.Apply(resource)
	}
}

type Composed []Patch

func (patches Composed) Apply(resource *unstructured.Unstructured) {
	for _, patch := range patches {
		patch.Apply(resource)
	}
}
