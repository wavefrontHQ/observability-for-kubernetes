package patch

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type ByName map[string]Patch

func (patches ByName) Apply(resource *unstructured.Unstructured) {
	patch := patches[resource.GetName()]
	if patch == nil {
		return
	}
	patch.Apply(resource)
}
