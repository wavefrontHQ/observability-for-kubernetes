package api

import (
	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
)

type CRSet struct {
	*wf.Wavefront
	*rc.ResourceCustomizations
}

func (c *CRSet) Spec() *SpecSet {
	var rcSpec *rc.ResourceCustomizationsSpec
	if c.ResourceCustomizations != nil {
		rcSpec = &c.ResourceCustomizations.Spec
	}
	return &SpecSet{
		WavefrontSpec:              &c.Wavefront.Spec,
		ResourceCustomizationsSpec: rcSpec,
	}
}

type SpecSet struct {
	*wf.WavefrontSpec
	*rc.ResourceCustomizationsSpec
}
