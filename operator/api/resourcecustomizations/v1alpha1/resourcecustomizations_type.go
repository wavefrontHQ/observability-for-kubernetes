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
	All    []ResourceCustomization          `json:"all,omitempty"`
	ByName map[string]WorkloadCustomization `json:"by_name,omitempty"`
}

// ResourceCustomization is a customization that can be applied to any and all kubernetes resources
type ResourceCustomization struct {
	Tolerations Tolerations `json:"tolerations,omitempty"`
}

// WorkloadCustomization is a customization that can only be applied to resources that have pods
type WorkloadCustomization struct {
	ResourceCustomization `json:",inline"`
	Resources             Resources `json:"resources,omitempty"`
}

type Tolerations struct {
	Add    []corev1.Toleration `json:"add,omitempty"`
	Remove []corev1.Toleration `json:"remove,omitempty"`
}

type Resources struct {
	// Requests CPU and Memory requirements
	Requests Resource `json:"requests,omitempty" yaml:"requests,omitempty"`

	// Limits CPU and Memory requirements
	Limits Resource `json:"limits,omitempty" yaml:"limits,omitempty"`
}

type Resource struct {
	// CPU is for specifying CPU requirements
	// +kubebuilder:validation:Pattern:=`^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$`
	CPU string `json:"cpu,omitempty" yaml:"cpu,omitempty"`

	// Memory is for specifying Memory requirements
	// +kubebuilder:validation:Pattern:=`^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$`
	Memory string `json:"memory,omitempty" yaml:"memory,omitempty"`

	// Memory is for specifying Memory requirements
	// +kubebuilder:validation:Pattern:=`^(\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))(([KMGTPE]i)|[numkMGTPE]|([eE](\+|-)?(([0-9]+(\.[0-9]*)?)|(\.[0-9]+))))?$`
	EphemeralStorage string `json:"ephemeral-storage,omitempty" yaml:"ephemeral-storage,omitempty"`
}
