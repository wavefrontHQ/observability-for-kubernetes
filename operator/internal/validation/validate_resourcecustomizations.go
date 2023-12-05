package validation

import (
	"fmt"

	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/result"
)

func ValidateRC(rcCR *rc.ResourceCustomizations) result.Result {
	for name, customization := range rcCR.Spec.ByName {
		if customization.Resources.IsEmpty() {
			continue
		}
		result := ValidateContainerResources(&customization.Resources, fmt.Sprintf("spec.byName.%s", name))
		if !result.IsValid() {
			return result
		}
	}
	return result.Valid
}
