package patch

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Composed []Patch

func (patches Composed) Apply(resource *unstructured.Unstructured) {
	for _, patch := range patches {
		if patch == nil {
			continue
		}
		patch.Apply(resource)
	}
}
