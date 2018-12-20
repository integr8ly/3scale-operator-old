package e2e

import (
	goctx "context"
	"github.com/integr8ly/3scale-operator/pkg/apis"
	"github.com/integr8ly/3scale-operator/pkg/apis/threescale/v1alpha1"
	routev1 "github.com/openshift/client-go/route/clientset/versioned/typed/route/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
	"time"
)

func TestThreescale(t *testing.T) {
	registerScheme(t)

	// run subtests
	t.Run("deployment", func(t *testing.T) {
		t.Run("basic", ThreescaleBasicDeployment)
	})
}

func ThreescaleBasicDeployment(t *testing.T) {

	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()

	err := ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}

	// get namespace
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	// get global framework variables
	f := framework.Global

	err = WaitForDeployment(t, f.KubeClient, namespace, "3scale-operator", 1, time.Second*10, time.Second*30)
	if err != nil {
		t.Fatal(err)
	}

	err = createDeployment(t, f, ctx)
	if err != nil {
		t.Fatalf("Failed to create deployment: %v", err)
	}

	err = verifyDeployment(t, f, ctx)
	if err != nil {
		t.Fatalf("Failed to verify deployment: %v", err)
	}
}

func registerScheme(t *testing.T) {
	threescaleList := &v1alpha1.ThreeScaleList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ThreeScale",
			APIVersion: "threescale.net/v1alpha1",
		},
	}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, threescaleList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}
}

func createDeployment(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}
	return f.Client.Create(goctx.TODO(), exampleThreeScale(namespace), &framework.CleanupOptions{TestContext: ctx, Timeout: time.Second * 10, RetryInterval: time.Second * 5})
}

func verifyDeployment(t *testing.T, f *framework.Framework, ctx *framework.TestCtx) error {
	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	systemDcs := []string{"system-mysql", "system-memcache", "system-redis", "system-resque", "system-sidekiq", "system-sphinx", "system-app"}
	backendDcs := []string{"backend-cron", "backend-listener", "backend-redis", "backend-worker"}
	zyncDcs := []string{"zync", "zync-database"}
	apiCastDcs := []string{"apicast-wildcard-router", "apicast-staging", "apicast-production"}
	routes := []string{"api-apicast-production-route", "api-apicast-staging-route", "apicast-wildcard-router-route", "backend-route", "system-developer-route", "system-master-admin-route", "system-provider-admin"}
	secrets := []string{"admin-credentials", "3scale-master-access-token"}

	err = verifyDeploymentConfigs(t, f, namespace, systemDcs)
	if err != nil {
		return err
	}
	err = verifyDeploymentConfigs(t, f, namespace, backendDcs)
	if err != nil {
		return err
	}
	err = verifyDeploymentConfigs(t, f, namespace, zyncDcs)
	if err != nil {
		return err
	}
	err = verifyDeploymentConfigs(t, f, namespace, apiCastDcs)
	if err != nil {
		return err
	}

	err = verifyRoutes(t, f, namespace, routes)
	if err != nil {
		return err
	}
	err = verifySecrets(t, f, namespace, secrets)
	if err != nil {
		return err
	}

	return nil
}

func verifySecrets(t *testing.T, f *framework.Framework, namespace string, secrets []string) error {
	for _, name := range secrets {
		_, err := f.KubeClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		t.Logf("secret'%s' available\n", name)
	}
	return nil
}

func verifyRoutes(t *testing.T, f *framework.Framework, namespace string, routes []string) error {
	osClient, err := routev1.NewForConfig(f.KubeConfig)
	if err != nil {
		return err
	}
	for _, name := range routes {
		_, err := osClient.Routes(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}
		t.Logf("route'%s' available\n", name)
	}
	return nil
}

func verifyDeploymentConfigs(t *testing.T, f *framework.Framework, namespace string, dcs []string) error {
	for _, dcname := range dcs {
		rcName := dcname + "-1"
		err := WaitForReplicationController(t, f.KubeClient, namespace, rcName, 1, time.Second*5, time.Minute*2)
		if err != nil {
			return err
		}
	}
	return nil
}
func exampleThreeScale(ns string) *v1alpha1.ThreeScale {
	return &v1alpha1.ThreeScale{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ThreeScale",
			APIVersion: "threescale.net/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "3scale-test",
			Namespace: ns,
		},
		Spec: v1alpha1.ThreeScaleSpec{
			RouteSuffix:   "example.com",
			Users:         []v1alpha1.ThreeScaleUser{},
			AuthProviders: []v1alpha1.ThreeScaleAuthProvider{},
		},
	}
}
