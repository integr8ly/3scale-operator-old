package main

import (
	"context"
	"runtime"
	"time"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	sdkVersion "github.com/operator-framework/operator-sdk/version"

	"flag"
	"os"

	"github.com/integr8ly/3scale-operator/pkg/apis/integreatly/v1alpha1"
	"github.com/integr8ly/3scale-operator/pkg/clients/openshift"
	"github.com/integr8ly/3scale-operator/pkg/stub"
	"github.com/integr8ly/3scale-operator/pkg/threescale"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/clientcmd"
	// Load Openshift types
	_ "github.com/integr8ly/3scale-operator/pkg/apis/openshift"
)

var (
	cfg v1alpha1.Config
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
	logrus.Infof("3Scale Version: %v", v1alpha1.ThreescaleVersion)
}

func init() {
	flagset := flag.CommandLine
	flagset.IntVar(&cfg.ResyncPeriod, "resync", 60, "change the resync period")
	flagset.StringVar(&cfg.LogLevel, "log-level", logrus.Level.String(logrus.InfoLevel), "Log level to use. Possible values: panic, fatal, error, warn, info, debug")
	flagset.Parse(os.Args[1:])
}

func main() {
	logLevel, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		logrus.Errorf("Failed to parse log level: %v", err)
	} else {
		logrus.SetLevel(logLevel)
	}
	printVersion()

	resource := v1alpha1.Group + "/" + v1alpha1.Version
	kind := v1alpha1.ThreescaleKind
	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		logrus.Fatalf("failed to get watch namespace: %v", err)
	}

	// Instantiate loader for kubeconfig file.
	kubeconfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	clientCfg, err := kubeconfig.ClientConfig()
	if err != nil {
		errors.Wrap(err, "error creating kube client config")
	}
	k8sClient := k8sclient.GetKubeClient()
	osClient := openshift.NewClientFactory(clientCfg)
	tsFactory := &threescale.ThreeScaleFactory{SecretClient: k8sClient.CoreV1().Secrets(namespace)}

	resyncDuration := time.Second * time.Duration(cfg.ResyncPeriod)
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncDuration)
	sdk.Watch(resource, kind, namespace, resyncDuration)
	sdk.Handle(stub.NewHandler(k8sClient, osClient, tsFactory))
	sdk.Run(context.TODO())
}
