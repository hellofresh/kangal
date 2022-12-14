package cmd

import (
	"flag"
	"fmt"

	"go.opentelemetry.io/otel/metric/global"

	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
	kubernetesClient "k8s.io/client-go/kubernetes"

	"github.com/hellofresh/kangal/pkg/core/observability"
	"github.com/hellofresh/kangal/pkg/kubernetes"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/typed/loadtest/v1"
	"github.com/hellofresh/kangal/pkg/proxy"
	"github.com/hellofresh/kangal/pkg/report"
)

type proxyCmdOpts struct {
	kubeConfig      string
	masterURL       string
	maxLoadTestsRun int
}

// NewProxyCmd creates a new proxy command
func NewProxyCmd() *cobra.Command {
	opts := &proxyCmdOpts{}

	cmd := &cobra.Command{
		Use:     "proxy",
		Short:   "Run proxy for accepting API requests",
		Aliases: []string{"p"},
		RunE: func(cmd *cobra.Command, args []string) error {
			flag.Parse()

			var cfg proxy.Config
			if err := envconfig.Process("", &cfg); err != nil {
				return fmt.Errorf("could not load config from env: %w", err)
			}

			logger, _, err := observability.NewLogger(cfg.Logger)
			if err != nil {
				return fmt.Errorf("could not build logger instance: %w", err)
			}

			pe, err := prometheus.New()
			if err != nil {
				return fmt.Errorf("could not build prometheus exporter: %w", err)
			}

			k8sConfig, err := kubernetes.BuildClientConfig(opts.masterURL, opts.kubeConfig, cfg.KubeClientTimeout)
			if err != nil {
				return fmt.Errorf("building config from flags: %w", err)
			}

			kangalClientSet, err := loadTestV1.NewForConfig(k8sConfig)
			if err != nil {
				return fmt.Errorf("building kangal clientset: %w", err)
			}

			kubeClientSet, err := kubernetesClient.NewForConfig(k8sConfig)
			if err != nil {
				return fmt.Errorf("building kubernetes clientset: %w", err)
			}

			loadTestClient := kangalClientSet.LoadTests()
			kubeClient := kubernetes.NewClient(loadTestClient, kubeClientSet, logger)

			provider := metric.NewMeterProvider(metric.WithReader(pe), metric.WithResource(
				resource.NewSchemaless(semconv.ServiceNameKey.String("kangal-proxy"))))

			global.SetMeterProvider(provider)

			statsReporter, err := proxy.NewMetricsReporter(provider.Meter("proxy"), kubeClient)
			if err != nil {
				return fmt.Errorf("error getting stats client:  %w", err)
			}

			err = report.InitObjectStorageClient(cfg.Report)
			if err != nil {
				return fmt.Errorf("building reportingClient client: %w", err)
			}

			cfg.MaxLoadTestsRun = opts.maxLoadTestsRun
			cfg.MasterURL = opts.masterURL

			return proxy.RunServer(cfg, proxy.Runner{
				Exporter:      pe,
				KubeClient:    kubeClient,
				Logger:        logger,
				StatsReporter: statsReporter,
			})
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&opts.kubeConfig, "kubeconfig", "", "absolute path to the kubernetes config")
	flags.StringVar(&opts.masterURL, "master-url", "", "The address of the Kubernetes API server. Overrides any value in kubeConfig. Only required if out-of-cluster.")
	flags.IntVar(&opts.maxLoadTestsRun, "max-load-tests", 10, "The maximum amount of load tests to run simultaneously.")
	return cmd
}
