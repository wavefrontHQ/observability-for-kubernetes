package validation

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/api"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/result"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Validate(objClient client.Client, crSet *api.CRSet) result.Aggregate {
	return result.Aggregate{
		crSet.Wavefront.GroupVersionKind():              ValidateWF(objClient, &crSet.Wavefront),
		crSet.ResourceCustomizations.GroupVersionKind(): ValidateRC(&crSet.ResourceCustomizations),
	}
}
