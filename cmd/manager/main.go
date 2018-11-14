package main

import (
	"flag"
	log "github.com/sirupsen/logrus"
	"os"
	"runtime"

	"github.com/integr8ly/3scale-operator/pkg/apis"
	threescalev1alpha1 "github.com/integr8ly/3scale-operator/pkg/apis/threescale/v1alpha1"
	"github.com/integr8ly/3scale-operator/pkg/controller"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var (
	cfg threescalev1alpha1.Config
)

func printVersion() {
	log.Printf("Go Version: %s", runtime.Version())
	log.Printf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	log.Printf("operator-sdk Version: %v", sdkVersion.Version)
	log.Infof("3Scale Version: %v", threescalev1alpha1.ThreescaleVersion)
}

func init() {
	flagset := flag.CommandLine
	flagset.IntVar(&cfg.ResyncPeriod, "resync", 60, "change the resync period")
	flagset.StringVar(&cfg.LogLevel, "log-level", log.Level.String(log.InfoLevel), "Log level to use. Possible values: panic, fatal, error, warn, info, debug")
	flagset.Parse(os.Args[1:])
}

func main() {
	logLevel, err := log.ParseLevel(cfg.LogLevel)
	if err != nil {
		log.Errorf("Failed to parse log level: %v", err)
	} else {
		log.SetLevel(logLevel)
	}
	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Fatalf("failed to get watch namespace: %v", err)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Fatal(err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Fatal(err)
	}

	log.Print("Starting the Cmd.")

	// Start the Cmd
	log.Fatal(mgr.Start(signals.SetupSignalHandler()))
}
