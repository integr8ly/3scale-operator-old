package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ThreeScaleTenantSpec defines the desired state of ThreeScaleTenant
type ThreeScaleTenantSpec struct {
	Name             string                   `json:"name"`
	AdminUsername    string                   `json:"adminUsername"`
	AdminEmail       string                   `json:"adminEmail"`
	AdminCredentials string                   `json:"adminCredentials"`
	Users            []ThreeScaleUser         `json:"users"`
	SeedUsers        SeedUsersConfig          `json:"seedUsers"`
	AuthProviders    []ThreeScaleAuthProvider `json:"authProviders"`
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

func (ts *ThreeScaleTenant) Defaults() {
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

func (ts *ThreeScaleTenant) Validate() error {
	return nil
}

func init() {
	SchemeBuilder.Register(&ThreeScaleTenant{}, &ThreeScaleTenantList{})
}
