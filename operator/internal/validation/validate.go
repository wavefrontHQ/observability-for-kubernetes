package validation

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/api"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Validate(objClient client.Client, crSet *api.CRSet) AggregateResult {
	return AggregateResult{
		crSet.Wavefront.GroupVersionKind():              ValidateWF(objClient, &crSet.Wavefront),
		crSet.ResourceCustomizations.GroupVersionKind(): ValidateRC(&crSet.ResourceCustomizations),
	}
}

type AggregateResult map[schema.GroupVersionKind]Result

func (ar AggregateResult) HasErrors() bool {
	for _, result := range ar {
		if result.IsError() {
			return true
		}
	}
	return false
}
