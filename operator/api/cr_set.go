package api

import (
	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
	wf "github.com/wavefronthq/observability-for-kubernetes/operator/api/wavefront/v1alpha1"
)

type CRSet struct {
	Wavefront              wf.Wavefront
	ResourceCustomizations rc.ResourceCustomizations
}

func (c *CRSet) Spec() *SpecSet {
	return &SpecSet{
		WavefrontSpec:              c.Wavefront.Spec,
		ResourceCustomizationsSpec: c.ResourceCustomizations.Spec,
	}
}

type SpecSet struct {
	wf.WavefrontSpec
	rc.ResourceCustomizationsSpec
}
