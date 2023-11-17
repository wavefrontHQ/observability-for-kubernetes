package patch

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type ApplyFn func(resource *unstructured.Unstructured)

func (a ApplyFn) Apply(resource *unstructured.Unstructured) {
	a(resource)
}
