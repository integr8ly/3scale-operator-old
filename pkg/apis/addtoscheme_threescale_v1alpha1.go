package apis

import (
	"github.com/integr8ly/3scale-operator/pkg/apis/threescale/v1alpha1"
	apps "github.com/openshift/api/apps/v1"
	authorization "github.com/openshift/api/authorization/v1"
	build "github.com/openshift/api/build/v1"
	image "github.com/openshift/api/image/v1"
	route "github.com/openshift/api/route/v1"
	template "github.com/openshift/api/template/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes,
		v1alpha1.SchemeBuilder.AddToScheme,
		apps.AddToScheme,
		authorization.AddToScheme,
		build.AddToScheme,
		image.AddToScheme,
		route.AddToScheme,
		template.AddToScheme,
	)
}
