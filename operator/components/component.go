package components

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Component interface {
	//TODO break this into two functions, Preprocess and Validate both return a validation.Result
	PreprocessAndValidate() validation.Result
	Resources() (resourcesToApply []client.Object, resourcesToDelete []client.Object, error error)
	//TODO find a better name for this function?
	TemplateDirectory() string
	Name() string
}

const DeployDir = "../deploy/components"
