package validation

import (
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func Validate(objClient client.Client, wavefront *wf.Wavefront) Result {
	return ValidateWF(objClient, wavefront)
}
