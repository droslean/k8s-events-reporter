package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/sirupsen/logrus"

	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/droslean/events-reporter/pkg/config"
	"github.com/droslean/events-reporter/pkg/controller"
	"github.com/droslean/events-reporter/pkg/scheduler"
)

type options struct {
	configPath string
}

func gatherOptions() options {
	o := options{}
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	fs.StringVar(&o.configPath, "config-path", "", "Path to config.yaml")
	fs.Parse(os.Args[1:])
	return o
}

func validateOptions(o options) error {
	if len(o.configPath) == 0 {
		return fmt.Errorf("--config-path was not provided")
	}
	return nil
}

func main() {
	o := gatherOptions()
	err := validateOptions(o)
	if err != nil {
		logrus.WithError(err).Fatal("invalid options")
	}
	clusterConfig, err := loadClusterConfig()
	if err != nil {
		logrus.WithError(err).Fatal("failed to load cluster config")
	}

	client, err := clientset.NewForConfig(clusterConfig)
	if err != nil {
		logrus.WithError(err).Fatal("failed to initialize kubernetes client")
	}

	config, err := config.NewConfig(o.configPath)
	if err != nil {
		logrus.WithError(err).Fatal("failed to load config")
	}
	stop := make(chan struct{})
	wg := &sync.WaitGroup{}

	// Start the controller
	controller := controller.NewController(config.EmailSettings)
	wg.Add(1)
	go controller.Start(stop, wg)

	// Start the schedulers
	for name, report := range config.Reports {
		s := scheduler.NewScheduler(name, report, client.CoreV1(), controller.Receiver)
		wg.Add(1)
		go s.Start(stop, wg)
	}

	c := make(chan os.Signal, 2)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	select {
	case <-c:
		logrus.Warn("Event reporter is terminating...")
		close(stop)
	}

	logrus.Info("Exiting...")
	wg.Wait()
}

func loadClusterConfig() (*rest.Config, error) {
	clusterConfig, err := rest.InClusterConfig()
	if err == nil {
		return clusterConfig, nil
	}

	credentials, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
	if err != nil {
		return nil, fmt.Errorf("could not load credentials from config: %v", err)
	}

	clusterConfig, err = clientcmd.NewDefaultClientConfig(*credentials, &clientcmd.ConfigOverrides{}).ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("could not load client configuration: %v", err)
	}
	return clusterConfig, nil
}
