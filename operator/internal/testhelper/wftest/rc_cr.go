package wftest

import (
	rc "github.com/wavefronthq/observability-for-kubernetes/operator/api/resourcecustomizations/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RCCROption func(rc *rc.ResourceCustomizations)

func RCCR(options ...RCCROption) *rc.ResourceCustomizations {
	cr := &rc.ResourceCustomizations{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ResourceCustomizations",
			APIVersion: rc.GroupVersion.Identifier(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-customizations",
			Namespace: DefaultNamespace,
		},
		Spec: rc.ResourceCustomizationsSpec{
			ByName: map[string]rc.WorkloadCustomization{},
		},
	}
	for _, option := range options {
		option(cr)
	}
	return cr
}
