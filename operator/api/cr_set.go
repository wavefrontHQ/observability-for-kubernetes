package api

import (
	ops "github.com/wavefronthq/observability-for-kubernetes/operator/api/operator_settings/v1alpha1"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
)

type CRSet struct {
	wf.Wavefront
	ops.OperatorSettings
}

func (c *CRSet) Spec() *SpecSet {
	return &SpecSet{
		WavefrontSpec:       c.Wavefront.Spec,
		OperatorSettingSpec: c.OperatorSettings.Spec,
	}
}

type SpecSet struct {
	wf.WavefrontSpec
	ops.OperatorSettingSpec
}
