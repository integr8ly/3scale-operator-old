package openshift

import (
	appsv1 "github.com/openshift/client-go/apps/clientset/versioned/typed/apps/v1"
	authv1 "github.com/openshift/client-go/authorization/clientset/versioned/typed/authorization/v1"
	imagev1 "github.com/openshift/client-go/image/clientset/versioned/typed/image/v1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	"k8s.io/client-go/rest"
)

func NewClientFactory(cfg *rest.Config) *ClientFactory {
	return &ClientFactory{cfg: cfg}
}

type ClientFactory struct {
	cfg *rest.Config
}

func (c *ClientFactory) AuthClient() (*authv1.AuthorizationV1Client, error) {
	return authv1.NewForConfig(c.cfg)
}

func (c *ClientFactory) ImageStreamClient() (*imagev1.ImageV1Client, error) {
	return imagev1.NewForConfig(c.cfg)
}

func (c *ClientFactory) AppsClient() (*appsv1.AppsV1Client, error) {
	return appsv1.NewForConfig(c.cfg)
}

func (c *ClientFactory) RouteClient() (*routev1.RouteV1Client, error) {
	return routev1.NewForConfig(c.cfg)
}
