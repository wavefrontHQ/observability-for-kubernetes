package components

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/internal/validation"
)

type Component interface {
	PreprocessAndValidate() validation.Result
	ReadAndDeleteResources() error
	ReadAndCreatResources() error
	//ReportHealthStatus() wf.WavefrontStatus
}
