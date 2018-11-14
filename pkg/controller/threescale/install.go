package threescale

import (
	"github.com/gobuffalo/packr"
	openshift "github.com/integr8ly/3scale-operator/pkg/apis/openshift/client"
	threescalev1alpha1 "github.com/integr8ly/3scale-operator/pkg/apis/threescale/v1alpha1"
	"github.com/openshift/api/template/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (r *ReconcileThreeScale) GetInstallResourcesAsRuntimeObjects(threescale *threescalev1alpha1.ThreeScale, params map[string]string) ([]runtime.Object, error) {
	rawExtensions, err := r.GetInstallResources(threescale, params)
	if err != nil {
		return nil, err
	}

	objects := make([]runtime.Object, 0)

	for _, rawObj := range rawExtensions {
		res, err := openshift.LoadKubernetesResource(rawObj.Raw)
		if err != nil {
			return nil, err

		}
		objects = append(objects, res)
	}

	return objects, nil
}

func (r *ReconcileThreeScale) GetInstallResources(threescale *threescalev1alpha1.ThreeScale, params map[string]string) ([]runtime.RawExtension, error) {
	templatesBox := packr.NewBox("../../../deploy/templates")
	res, err := openshift.LoadKubernetesResourceFromBox(templatesBox, threescalev1alpha1.TEMPLATE_NAME)
	if err != nil {
		return nil, err
	}

	templ := res.(*v1.Template)
	processor, err := openshift.NewTemplate(threescale.Namespace, r.config, openshift.TemplateDefaultOpts)
	if err != nil {
		return nil, err
	}

	return processor.Process(templ, params, openshift.TemplateDefaultOpts)
}
