package v1alpha1

import (
	"github.com/wavefronthq/observability-for-kubernetes/operator/api/common"
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
// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// ResourceCustomizations is the Schema for the resourcecustomiztaions API
type ResourceCustomizations struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ResourceCustomizationsSpec   `json:"spec,omitempty"`
	Status ResourceCustomizationsStatus `json:"status,omitempty"`
}

type ResourceCustomizationsSpec struct {
	All    *ResourceCustomization           `json:"all,omitempty"`
	ByName map[string]WorkloadCustomization `json:"byName,omitempty"`
}

// ResourceCustomization is a customization that can be applied to any and all kubernetes resources
type ResourceCustomization struct {
	Tolerations Tolerations `json:"tolerations,omitempty"`
}

// WorkloadCustomization is a customization that can only be applied to resources that have pods
type WorkloadCustomization struct {
	ResourceCustomization `json:",inline"`
	Resources             common.ContainerResources `json:"resources,omitempty"`
}

type Tolerations struct {
	Add    []corev1.Toleration `json:"add,omitempty"`
	Remove []corev1.Toleration `json:"remove,omitempty"`
}

// ResourceCustomizationsStatus reports validation errors for the ResourceCustomizations CR
type ResourceCustomizationsStatus struct {
	// Message is a human-readable message indicating details about any validation errors.
	Message string `json:"message,omitempty"`
}
