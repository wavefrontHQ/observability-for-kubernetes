package validation

import (
	"fmt"

	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/result"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/errors"
)

func ValidateTolerations(tolerations []v1.Toleration, resourceName string) result.Result {
	var errs []error
	for _, toleration := range tolerations {
		if len(toleration.Operator) > 0 && toleration.Operator != v1.TolerationOpEqual && toleration.Operator != v1.TolerationOpExists {
			errs = append(errs, fmt.Errorf("invalid %s.toleration: toleration with key %s must have operator value of %s or %s", resourceName, toleration.Key, v1.TolerationOpEqual, v1.TolerationOpExists))
		}

		if toleration.Effect != v1.TaintEffectNoSchedule && toleration.Effect != v1.TaintEffectNoExecute && toleration.Effect != v1.TaintEffectPreferNoSchedule {
			errs = append(errs, fmt.Errorf("invalid %s.toleration: toleration with key %s must have effect value of %s, %s, or %s", resourceName, toleration.Key, v1.TaintEffectNoExecute, v1.TaintEffectNoSchedule, v1.TaintEffectPreferNoSchedule))
		}

	}
	err := errors.NewAggregate(errs)
	if err != nil {
		return result.NewError(errors.NewAggregate(errs))
	}
	return result.Valid
}
