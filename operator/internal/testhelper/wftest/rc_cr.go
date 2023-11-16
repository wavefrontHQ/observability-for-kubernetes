package wftest

import (
	ops "github.com/wavefronthq/observability-for-kubernetes/operator/api/operator_settings/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type RCCROption func(rc *ops.OperatorSettings)

func RCCr(options ...RCCROption) *ops.OperatorSettings {
	cr := &ops.OperatorSettings{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "resource-customizations",
			Namespace: DefaultNamespace,
		},
		Spec: ops.OperatorSettingSpec{
			ByName: map[string]ops.ResourceCustomization{},
		},
	}
	for _, option := range options {
		option(cr)
	}
	return cr
}
