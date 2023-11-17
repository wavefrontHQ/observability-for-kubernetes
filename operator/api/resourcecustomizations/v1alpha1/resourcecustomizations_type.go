package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&ResourceCustomizations{}, &ResourceCustomizationsList{})
}

// +kubebuilder:object:root=true
// ResourceCustomizationsList contains a list of ResourceCustomizations
type ResourceCustomizationsList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ResourceCustomizations `json:"items"`
}

// +kubebuilder:object:root=true
// ResourceCustomizations is the Schema for the resourcecustomiztaions API
type ResourceCustomizations struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ResourceCustomizationsSpec `json:"spec,omitempty"`
}

type ResourceCustomizationsSpec struct {
	ByName ByName `json:"by_name,omitempty"`
}

type ByName map[string]ResourceCustomization

type TolerationsCustomization struct {
	Add []corev1.Toleration `json:"add,omitempty"`
}

type ResourceCustomization struct {
	Tolerations TolerationsCustomization `json:"tolerations,omitempty"`
}
