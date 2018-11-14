package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ThreeScaleTenantSpec defines the desired state of ThreeScaleTenant
type ThreeScaleTenantSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// ThreeScaleTenantStatus defines the observed state of ThreeScaleTenant
type ThreeScaleTenantStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
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
	SchemeBuilder.Register(&ThreeScaleTenant{}, &ThreeScaleTenantList{})
}
