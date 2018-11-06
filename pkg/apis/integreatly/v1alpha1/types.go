package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Group             = "3scale.net"
	Version           = "v1alpha1"
	ThreescaleKind    = "ThreeScale"
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
	Namespace         string                   `json:"namespace"`
	AdminCredentials  string                   `json:"adminCredentials"`
	MasterCredentials string                   `json:"masterCredentials"`
	RouteSuffix       string                   `json:"routeSuffix"`
	RWXStorageClass   string                   `json:"rwxStorageClass"`
	AdminUsername     string                   `json:"adminUsername"`
	AdminEmail        string                   `json:"adminEmail"`
	Users             []ThreeScaleUser         `json:"users"`
	SeedUsers         SeedUsersConfig          `json:"seedUsers"`
	AuthProviders     []ThreeScaleAuthProvider `json:"authProviders"`
	WildcardPolicy    string                   `json:"wildcardPolicy"`
	MysqlPvcSize      string                   `json:"mysqlPvcSize"`
}

type SeedUsersConfig struct {
	Count       int    `json:"count,omitempty"`
	EmailFormat string `json:"emailFormat,omitempty"`
	NameFormat  string `json:"nameFormat,omitempty"`
	Password    string `json:"password,omitempty"`
	Role        string `json:"role,omitempty"`
}

type ThreeScaleUserList struct {
	Items []*ThreeScaleApiUser `json:"users"`
}

type ThreeScaleApiUser struct {
	User *ThreeScaleUser `json:"user"`
}

type ThreeScaleUser struct {
	ID       int    `json:"id,omitempty"`
	UserName string `json:"username,omitempty"`
	Email    string `json:"email,omitempty"`
	Password string `json:"password,omitempty"`
	Role     string `json:"role,omitempty"`
	State    string `json:"state,omitempty"`
}

type ThreeScaleAuthProviderList struct {
	Items []*ThreeScaleApiAuthProvider `json:"authentication_providers"`
}

type ThreeScaleApiAuthProvider struct {
	AuthProvider *ThreeScaleAuthProvider `json:"authentication_provider"`
}

type ThreeScaleAuthProvider struct {
	ID                             int    `json:"id,omitempty"`
	Kind                           string `json:"kind,omitempty"`
	Name                           string `json:"name,omitempty"`
	ClientID                       string `json:"client_id,omitempty"`
	ClientSecret                   string `json:"client_secret,omitempty"`
	Site                           string `json:"site,omitempty"`
	SkipSslCertificateVerification bool   `json:"skip_ssl_certificate_verification,omitempty"`
	Published                      bool   `json:"published,omitempty"`
}

type ThreeScaleUserPair struct {
	TsUser   *ThreeScaleUser
	SpecUser *ThreeScaleUser
}

type ThreeScaleAuthProviderPair struct {
	TsAuthProvider   *ThreeScaleAuthProvider
	SpecAuthProvider *ThreeScaleAuthProvider
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
	if ts.Spec.SeedUsers.Role == "" {
		ts.Spec.SeedUsers.Role = "admin"
	}
	if ts.Spec.SeedUsers.EmailFormat == "" {
		ts.Spec.SeedUsers.EmailFormat = "evals%02d@example.com"
	}
	if ts.Spec.SeedUsers.NameFormat == "" {
		ts.Spec.SeedUsers.NameFormat = ts.Spec.SeedUsers.EmailFormat
	}
	if ts.Spec.SeedUsers.Password == "" {
		ts.Spec.SeedUsers.Password = "Password1"
	}
}

func (ts *ThreeScale) Validate() error {
	return nil
}
