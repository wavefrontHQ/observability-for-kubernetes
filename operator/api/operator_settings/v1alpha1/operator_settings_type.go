package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

// +kubebuilder:object:root=true
// OperatorSettings is the Schema for the operator API
type OperatorSettings struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec OperatorSettingSpec `json:"spec,omitempty"`
}

type OperatorSettingSpec struct {
	DoesItWork bool `json:"does_it_work"`
}
