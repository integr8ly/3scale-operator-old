package stub

import (
	"github.com/gobuffalo/packr"
	"github.com/integr8ly/3scale-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/3scale-operator/pkg/apis/openshift/template"
	"github.com/integr8ly/3scale-operator/pkg/util"
	"github.com/openshift/api/template/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func GetInstallResourcesAsRuntimeObjects(threescale *v1alpha1.ThreeScale, params map[string]string) ([]runtime.Object, error) {
	rawExtensions, err := GetInstallResources(threescale, params)
	if err != nil {
		return nil, err
	}

	objects := make([]runtime.Object, 0)

	for _, rawObj := range rawExtensions {
		res, err := util.LoadKubernetesResource(rawObj.Raw)
		if err != nil {
			return nil, err

		}
		objects = append(objects, res)
	}

	return objects, nil
}

func GetInstallResources(threescale *v1alpha1.ThreeScale, params map[string]string) ([]runtime.RawExtension, error) {
	templatesBox := packr.NewBox("../../deploy/templates")

	res, err := util.LoadKubernetesResourceFromBox(templatesBox, v1alpha1.TEMPLATE_NAME)
	if err != nil {
		return nil, err
	}

	templ := res.(*v1.Template)
	processor, err := template.NewTemplateProcessor(threescale.Namespace)
	if err != nil {
		return nil, err
	}

	return processor.Process(templ, params)
}
