package client

import (
	"bytes"
	"encoding/json"
	v1template "github.com/openshift/api/template/v1"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubescheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/rest/fake"
	"net/http"
	"path"
	"testing"
)

func TestNewTemplate(t *testing.T) {
	cases := []struct {
		Name        string
		Namespace   string
		Client      func() *rest.Config
		Opts        TemplateOpt
		Validate    func(tmpl *Template, t *testing.T)
		ExpectError bool
	}{
		{
			Name:      "Should create template ref",
			Namespace: "test",
			Client: func() *rest.Config {
				return &rest.Config{}
			},
			Opts: TemplateDefaultOpts,
			Validate: func(tmpl *Template, t *testing.T) {
				if tmpl.getNS() != "test" {
					t.Fatalf("Invalid template namespace: %v", tmpl)
				}
			},
			ExpectError: false,
		},
	}

	for _, tc := range cases {
		tmpl, err := NewTemplate(tc.Namespace, tc.Client(), tc.Opts)

		if tc.ExpectError && err == nil {
			t.Fatalf("expected an error but got none")
		}

		if !tc.ExpectError && err != nil {
			t.Fatalf("did not expect error but got %s ", err)
		}

		tc.Validate(tmpl, t)
	}
}

func objBody(object interface{}) io.ReadCloser {
	output, err := json.MarshalIndent(object, "", "")
	if err != nil {
		panic(err)
	}
	return ioutil.NopCloser(bytes.NewReader([]byte(output)))
}

func TestTemplate_Process(t *testing.T) {
	cases := []struct {
		Name         string
		Namespace    string
		Params       map[string]string
		templatePath string
		StatusCode   int
		SendErr      error
		ExpectError  bool
		Client       func(tp string, statusCode int, sendError error) *fake.RESTClient
	}{
		{
			Name:      "Should process template",
			Namespace: "test",
			Params: map[string]string{
				"OPENSHIFT_HOST": "127.0.0.1:8443",
				"SSO_HOST":       "sso.127.0.0.1:8443",
			},
			templatePath: path.Join("_testdata", "test-template.yaml"),
			StatusCode:   201,
			SendErr:      nil,
			ExpectError:  false,
			Client: func(tp string, statusCode int, sendErr error) *fake.RESTClient {
				rawData, err := ioutil.ReadFile(tp)
				if err != nil {
					t.Fatalf("Could not find template file %v", err)
				}
				jsonData, err := JsonIfYaml(rawData, tp)
				if err != nil {
					t.Fatalf("Could not find template file %v", err)
				}

				serverVersions := []string{"/v1", "/templates"}
				fakeClient := &fake.RESTClient{
					NegotiatedSerializer: kubescheme.Codecs,
					Resp: &http.Response{
						StatusCode: statusCode,
						Body:       objBody(&metav1.APIVersions{Versions: serverVersions}),
					},
					Client: fake.CreateHTTPClient(func(req *http.Request) (*http.Response, error) {
						if sendErr != nil {
							return nil, sendErr
						}
						header := http.Header{}
						header.Set("Content-Type", "application/json")
						return &http.Response{StatusCode: statusCode, Header: header, Body: ioutil.NopCloser(bytes.NewReader(jsonData))}, nil
					}),
				}

				return fakeClient
			},
		},
	}

	for _, tc := range cases {
		tmpl := Template{
			namespace:  tc.Namespace,
			RestClient: tc.Client(tc.templatePath, tc.StatusCode, tc.SendErr),
		}
		template := v1template.Template{}

		_, err := tmpl.Process(&template, tc.Params, TemplateDefaultOpts)

		if tc.ExpectError && err == nil {
			t.Fatalf("expected an error but got none")
		}

		if !tc.ExpectError && err != nil {
			t.Fatalf("did not expect error but got %s ", err)
		}
	}
}

func TestTemplate_FillParams(t *testing.T) {
	cases := []struct {
		Name        string
		Namespace   string
		Client      func() *rest.Config
		Opts        TemplateOpt
		Params      map[string]string
		Validate    func(tmpl *v1template.Template, params map[string]string, t *testing.T)
		ExpectError bool
	}{
		{
			Name:      "Should fill template with params",
			Namespace: "test",
			Client: func() *rest.Config {
				return &rest.Config{}
			},
			Opts:   TemplateDefaultOpts,
			Params: map[string]string{"HOST": "localhost", "PORT": "8443"},
			Validate: func(tmpl *v1template.Template, params map[string]string, t *testing.T) {
				for _, v := range tmpl.Parameters {
					_, ok := params[v.Name]
					if !ok {
						t.Fatalf("Invalid param in template object: %s", v.Name)
					}
				}
			},
			ExpectError: false,
		},
		{
			Name:      "Should ignore empty params map",
			Namespace: "test",
			Client: func() *rest.Config {
				return &rest.Config{}
			},
			Opts:   TemplateDefaultOpts,
			Params: map[string]string{"HOST": "localhost", "PORT": "8443"},
			Validate: func(tmpl *v1template.Template, params map[string]string, t *testing.T) {
				for _, v := range tmpl.Parameters {
					_, ok := params[v.Name]
					if ok {
						t.Fatalf("Invalid param in template object: %s", v.Name)
					}
				}
			},
			ExpectError: false,
		},
	}

	for _, tc := range cases {
		tmpl := v1template.Template{}
		tmplEngine, err := NewTemplate("test", tc.Client(), tc.Opts)

		if tc.ExpectError && err == nil {
			t.Fatalf("expected an error but got none")
		}

		if !tc.ExpectError && err != nil {
			t.Fatalf("did not expect error but got %s ", err)
		}

		emptyParams := make(map[string]string)
		tmplEngine.FillParams(&tmpl, emptyParams)

		tc.Validate(&tmpl, tc.Params, t)
	}
}
