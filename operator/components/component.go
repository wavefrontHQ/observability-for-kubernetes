package components

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Component interface {
	PreprocessAndValidate() validation.Result
	Resources() (resourcesToApply []client.Object, resourcesToDelete []client.Object, error error)
	TemplateDirectory() string
	Name() string
}

const DeployDir = "../deploy/components"
