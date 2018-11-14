package v1alpha1

import (
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ThreeScaleTenantSpec defines the desired state of ThreeScaleTenant
type ThreeScaleTenantSpec struct {
	Name string `json:"name"`
}

// ThreeScaleTenantStatus defines the observed state of ThreeScaleTenant
type ThreeScaleTenantStatus struct {
	Ready bool `json:"ready"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ThreeScaleTenant is the Schema for the threescaletenants API
// +k8s:openapi-gen=true
type ThreeScaleTenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ThreeScaleTenantSpec   `json:"spec,omitempty"`
	Status ThreeScaleTenantStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ThreeScaleTenantList contains a list of ThreeScaleTenant
type ThreeScaleTenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ThreeScaleTenant `json:"items"`
}

func init() {
	fmt.Println("init threescale tenant types")

	SchemeBuilder.Register(&ThreeScaleTenant{}, &ThreeScaleTenantList{})
}
