package controller

import (
	"fmt"

	"contrib.go.opencensus.io/exporter/prometheus"
	"go.uber.org/zap"
	kubeInformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"

	"github.com/hellofresh/kangal/pkg/backends"
	"github.com/hellofresh/kangal/pkg/core/observability"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
	"github.com/hellofresh/kangal/pkg/kubernetes/generated/informers/externalversions"
)

// Runner encapsulates all Kangal Controller dependencies
type Runner struct {
	Exporter       *prometheus.Exporter
	Logger         *zap.Logger
	KubeClient     kubernetes.Interface
	KangalClient   *clientSetV.Clientset
	KubeInformer   kubeInformers.SharedInformerFactory
	KangalInformer externalversions.SharedInformerFactory
	StatsReporter  observability.StatsReporter
}

// Run runs an instance of kubernetes kubeController
func Run(cfg Config, rr Runner) error {
	stopCh := make(chan struct{})

	registry := backends.New(
		backends.WithLogger(rr.Logger),
		backends.WithKubeClientSet(rr.KubeClient),
		backends.WithKangalClientSet(rr.KangalClient),
		backends.WithNamespaceLister(rr.KubeInformer.Core().V1().Namespaces().Lister()),
		backends.WithPodAnnotations(cfg.PodAnnotations),
	)

	c := NewController(cfg, rr.KubeClient, rr.KangalClient, rr.KubeInformer, rr.KangalInformer, rr.StatsReporter, registry, rr.Logger)

	// notice that there is no need to run Start methods in a separate goroutine. (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	rr.KangalInformer.Start(stopCh)
	rr.KubeInformer.Start(stopCh)

	if err := RunMetricsServer(cfg, rr, stopCh); err != nil {
		return fmt.Errorf("could not initialise Metrics Server: %w", err)
	}

	if err := c.Run(1, stopCh); err != nil {
		return fmt.Errorf("error running kubeController: %w", err)
	}
	return nil
}
