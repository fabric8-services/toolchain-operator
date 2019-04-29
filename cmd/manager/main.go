package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/fabric8-services/toolchain-operator/pkg/apis"
	"github.com/fabric8-services/toolchain-operator/pkg/controller"
	"github.com/fabric8-services/toolchain-operator/pkg/online_registration"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/ready"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var log = logf.Log.WithName("cmd")

func printVersion() {
	log.Info(fmt.Sprintf("Go Version: %s", runtime.Version()))
	log.Info(fmt.Sprintf("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH))
	log.Info(fmt.Sprintf("operator-sdk Version: %v", sdkVersion.Version))
}

func main() {
	flag.Parse()

	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	// ToDo: Use Logrus
	logf.SetLogger(logf.ZapLogger(false))

	printVersion()

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Become the leader before proceeding
	if err := leader.Become(context.TODO(), "toolchain-enabler-lock"); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	r := ready.NewFileReady()
	err = r.Set()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	defer func() {
		if err := r.Unset(); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
	}()

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: namespace})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	secondaryCache, err := cache.New(mgr.GetConfig(), cache.Options{Namespace: online_registration.Namespace, Scheme: mgr.GetScheme(), Mapper: mgr.GetRESTMapper()})
	if err != nil {
		log.Error(fmt.Errorf("failed to create openshift-infra cache: %v", err), "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr, secondaryCache); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err := start(mgr, secondaryCache); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

}

func start(mgr manager.Manager, cache cache.Cache) error {
	stop := signals.SetupSignalHandler()
	errChan := make(chan error)

	go func() {
		// Start secondary cache for openshift-infra ns.
		if err := cache.Start(stop); err != nil {
			errChan <- err
		}
	}()

	log.Info("waiting for cache to sync")
	if !cache.WaitForCacheSync(stop) {
		return fmt.Errorf("failed to sync cache")
	}
	log.Info("cache synced")

	go func() {
		log.Info("Starting the Cmd.")
		errChan <- mgr.Start(stop)
	}()

	return <-errChan
}
