package patch

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

type Patch interface {
	Apply(resource *unstructured.Unstructured)
}
