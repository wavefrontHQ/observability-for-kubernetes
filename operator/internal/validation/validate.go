package validation

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/api"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Validate(objClient client.Client, crSet *api.CRSet) Result {
	result := ValidateWF(objClient, &crSet.Wavefront)
	if !result.IsValid() {
		return result
	}
	return ValidateRC(&crSet.ResourceCustomizations)
}
