package v1alpha1

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestThreeScaleTenant_Defaults(t *testing.T) {
	tests := []struct {
		name     string
		ts       ThreeScaleTenant
		validate func(t *testing.T, ts *ThreeScaleTenant)
	}{
		{
			name: "test empty seed user config",
			ts: ThreeScaleTenant{
				Spec: ThreeScaleTenantSpec{
					SeedUsers: SeedUsersConfig{},
				},
			},
			validate: func(t *testing.T, ts *ThreeScaleTenant) {
				if ts.Spec.SeedUsers.Count != 0 {
					t.Fatal("failed default seed users count should be 0")
				}
				if ts.Spec.SeedUsers.Role != "admin" {
					t.Fatal("failed default seed users role should be 'admin")
				}
				if ts.Spec.SeedUsers.Password != "Password1" {
					t.Fatal("failed default seed users password should be 'Password1'")
				}
				if ts.Spec.SeedUsers.EmailFormat != "evals%02d@example.com" {
					t.Fatal("failed default seed users email format should be 'evals02d@example.com'")
				}
				if ts.Spec.SeedUsers.NameFormat != "evals%02d@example.com" {
					t.Fatal("failed default seed users name format should be 'evals02d@example.com'")
				}
			},
		},
		{
			name: "test seed users config",
			ts: ThreeScaleTenant{
				Spec: ThreeScaleTenantSpec{
					SeedUsers: SeedUsersConfig{
						Count:       1,
						Role:        "member",
						NameFormat:  "test",
						EmailFormat: "test@example.com",
						Password:    "password",
					},
				},
			},
			validate: func(t *testing.T, ts *ThreeScaleTenant) {
				if ts.Spec.SeedUsers.Count != 1 {
					t.Fatal("failed seed users count should be 1")
				}
				if ts.Spec.SeedUsers.Role != "member" {
					t.Fatal("failed seed users role should be 'member")
				}
				if ts.Spec.SeedUsers.Password != "password" {
					t.Fatal("failed seed users password should be 'password'")
				}
				if ts.Spec.SeedUsers.EmailFormat != "test@example.com" {
					t.Fatal("failed seed users email format should be 'test@example.com'")
				}
				if ts.Spec.SeedUsers.NameFormat != "test" {
					t.Fatal("failed seed users name format should be 'test'")
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := tt.ts
			ts.Defaults()

			if tt.validate != nil {
				tt.validate(t, &ts)
			}
		})
	}
}

func TestThreeScaleTenant_Validate(t *testing.T) {
	type fields struct {
		TypeMeta   metav1.TypeMeta
		ObjectMeta metav1.ObjectMeta
		Spec       ThreeScaleTenantSpec
		Status     ThreeScaleTenantStatus
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := &ThreeScaleTenant{
				TypeMeta:   tt.fields.TypeMeta,
				ObjectMeta: tt.fields.ObjectMeta,
				Spec:       tt.fields.Spec,
				Status:     tt.fields.Status,
			}
			if err := ts.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("ThreeScaleTenant.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
