package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ThreescaleVersion = "2.2.0.GA"
	TEMPLATE_NAME     = "3scale-amp-2.2.0.GA.yml"
)

type Config struct {
	ResyncPeriod int
	LogLevel     string
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ThreeScaleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []ThreeScale `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ThreeScale struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              ThreeScaleSpec   `json:"spec"`
	Status            ThreeScaleStatus `json:"status,omitempty"`
}

type ThreeScaleSpec struct {
	Namespace         string `json:"namespace"`
	TenantName        string `json:"tenantName"`
	AdminCredentials  string `json:"adminCredentials"`
	MasterCredentials string `json:"masterCredentials"`
	RouteSuffix       string `json:"routeSuffix"`
	RWXStorageClass   string `json:"rwxStorageClass"`
	AdminUsername     string `json:"adminUsername"`
	AdminEmail        string `json:"adminEmail"`
	WildcardPolicy    string `json:"wildcardPolicy"`
	MysqlPvcSize      string `json:"mysqlPvcSize"`
}

type T interface{}

type StatusPhase string

type ThreeScaleStatus struct {
	Version string      `json:"version"`
	Phase   StatusPhase `json:"phase"`
	// marked as true when all work is done on it
	Ready bool `json:"ready"`
}

var (
	NoPhase                   StatusPhase = ""
	PhaseProvisionCredentials StatusPhase = "credentials"
	PhaseReconcileThreescale  StatusPhase = "reconcile"
)

func (ts *ThreeScale) Defaults() {
}

func (ts *ThreeScale) Validate() error {
	return nil
}

func init() {
	SchemeBuilder.Register(&ThreeScale{}, &ThreeScaleList{})
}
