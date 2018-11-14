package client

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	"testing"
)

func TestWatchNegotiatedSerializer_SupportedMediaTypes(t *testing.T) {
	cases := []struct {
		Name        string
		MediaTypes  func() []runtime.SerializerInfo
		Get         func(s []runtime.SerializerInfo) runtime.SerializerInfo
		Validate    func(m runtime.SerializerInfo) error
		ExpectError bool
	}{
		{
			Name: "Should validate json media type",
			MediaTypes: func() []runtime.SerializerInfo {
				w := watchNegotiatedSerializer{}
				return w.SupportedMediaTypes()
			},
			Get: func(s []runtime.SerializerInfo) runtime.SerializerInfo {
				return s[0]
			},
			Validate: func(m runtime.SerializerInfo) error {
				if m.MediaType != "application/json" {
					return fmt.Errorf("Invalid json media type: %s", m.MediaType)
				}

				return nil
			},
			ExpectError: false,
		},
	}

	for _, tc := range cases {
		mediaType := tc.Get(tc.MediaTypes())
		err := tc.Validate(mediaType)

		if !tc.ExpectError && err != nil {
			t.Fatalf("Case validation failed: %v", err)
		}

		if tc.ExpectError && err == nil {
			t.Fatalf("expected an error but got none")
		}
	}
}
