package client

import (
	"encoding/json"
	"fmt"
	v1template "github.com/openshift/api/template/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
)

func NewTemplate(namespace string, inConfig *rest.Config, opts TemplateOpt) (*Template, error) {
	config := rest.CopyConfig(inConfig)
	config.GroupVersion = &schema.GroupVersion{
		Group:   opts.ApiGroup,
		Version: opts.ApiVersion,
	}
	config.APIPath = opts.ApiPath
	config.AcceptContentTypes = opts.ApiMimetype
	config.ContentType = opts.ApiMimetype

	config.NegotiatedSerializer = basicNegotiatedSerializer{}
	if config.UserAgent == "" {
		config.UserAgent = rest.DefaultKubernetesUserAgent()
	}

	restClient, err := rest.RESTClientFor(config)
	if err != nil {
		return nil, err
	}

	return &Template{namespace: namespace, RestClient: restClient}, nil
}

func (template *Template) getNS() string {
	return template.namespace
}

func (template *Template) Process(tmpl *v1template.Template, params map[string]string, opts TemplateOpt) ([]runtime.RawExtension, error) {
	template.FillParams(tmpl, params)
	resource, err := json.Marshal(tmpl)
	if err != nil {
		return nil, err
	}

	result := template.RestClient.
		Post().
		Namespace(template.namespace).
		Body(resource).
		Resource(opts.ApiResource).
		Do()

	if result.Error() == nil {
		data, err := result.Raw()
		if err != nil {
			return nil, err
		}

		templ, err := LoadKubernetesResource(data)
		if err != nil {
			return nil, err
		}

		if v1Temp, ok := templ.(*v1template.Template); ok {
			return v1Temp.Objects, nil
		}

		return nil, fmt.Errorf("Wrong type returned by the server: %v", templ)
	}

	return nil, result.Error()
}

func (template *Template) FillParams(tmpl *v1template.Template, params map[string]string) {
	for i, param := range tmpl.Parameters {
		if value, ok := params[param.Name]; ok {
			tmpl.Parameters[i].Value = value
		}
	}
}
