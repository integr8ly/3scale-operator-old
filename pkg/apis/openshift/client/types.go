package client

import (
	v1template "github.com/openshift/api/template/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
)

var YamlExtensions = [2]string{
	"yml",
	"yaml",
}

type Template struct {
	namespace  string
	RestClient rest.Interface
}

type TemplateOpt struct {
	ApiKind     string
	ApiVersion  string
	ApiPath     string
	ApiGroup    string
	ApiMimetype string
	ApiResource string
}

var (
	TemplateDefaultOpts = TemplateOpt{
		ApiVersion:  "v1",
		ApiMimetype: "application/json",
		ApiPath:     "/apis",
		ApiGroup:    "template.openshift.io",
		ApiResource: "processedtemplates",
	}
)

type TemplateHandler interface {
	getNS() string
	Process(tmpl *v1template.Template, params map[string]string, opts TemplateOpt) ([]runtime.RawExtension, error)
	FillParams(tmpl *v1template.Template, params map[string]string)
}
