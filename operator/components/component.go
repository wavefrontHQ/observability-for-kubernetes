package components

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Component interface {
	//TODO: Component Refactor -  break this into two functions, Preprocess and Validate both return a validation.Result
	PreprocessAndValidate() validation.Result
	Resources() (resourcesToApply []client.Object, resourcesToDelete []client.Object, error error)
	Name() string
}

const DeployDir = "../deploy/components"
