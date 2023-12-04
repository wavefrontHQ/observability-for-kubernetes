package validation

import (
	"fmt"

	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
)

func ValidateRC(rcCR *rc.ResourceCustomizations) Result {
	for name, customization := range rcCR.Spec.ByName {
		if customization.Resources.IsEmpty() {
			continue
		}
		result := ValidateResources(&customization.Resources, fmt.Sprintf("spec.byName.%s", name))
		if !result.IsValid() {
			return result
		}
	}
	return Result{}
}
