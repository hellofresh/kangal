package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/hellofresh/kangal/pkg/core/observability"
	kube "github.com/hellofresh/kangal/pkg/kubernetes"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/typed/loadtest/v1"
	"github.com/hellofresh/kangal/pkg/proxy"
	"github.com/hellofresh/kangal/pkg/report"
)

type apiCmdOpts struct {
	kubeConfig      string
	masterURL       string
	maxLoadTestsRun int
}

// NewAPICmd creates a new api command
func NewAPICmd(ctx context.Context) *cobra.Command {
	opts := &apiCmdOpts{}

	cmd := &cobra.Command{
		Use:   "api",
		Short: "Run EXPERIMENTAL proxy for accepting API requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			flag.Parse()

			var cfg proxy.Config
			if err := envconfig.Process("", &cfg); err != nil {
				return fmt.Errorf("could not load config from env: %w", err)
			}

			logger, isDebug, err := observability.NewLogger(cfg.Logger)
			if err != nil {
				return fmt.Errorf("could not build logger instance: %w", err)
			}

			pe, err := observability.NewPrometheusExporter("kangal-proxy", nil)
			if err != nil {
				return fmt.Errorf("could not initialise Prometheus exporter: %w", err)
			}

			k8sConfig, err := clientcmd.BuildConfigFromFlags(opts.masterURL, opts.kubeConfig)
			if err != nil {
				return fmt.Errorf("building config from flags: %w", err)
			}

			kangalClientSet, err := loadTestV1.NewForConfig(k8sConfig)
			if err != nil {
				return fmt.Errorf("building kangal clientset: %w", err)
			}

			kubeClientSet, err := kubernetes.NewForConfig(k8sConfig)
			if err != nil {
				return fmt.Errorf("building kubernetes clientset: %w", err)
			}

			loadTestClient := kangalClientSet.LoadTests()
			kubeClient := kube.NewClient(loadTestClient, kubeClientSet, logger)

			err = report.InitObjectStorageClient(cfg.Report)
			if err != nil {
				return fmt.Errorf("building reportingClient client: %w", err)
			}

			cfg.MaxLoadTestsRun = opts.maxLoadTestsRun
			cfg.MasterURL = opts.masterURL

			return proxy.RunAPIServer(ctx, cfg, proxy.APIRunner{
				Config:     cfg.GRPC,
				Exporter:   pe,
				KubeClient: kubeClient,
				Logger:     logger,
				Debug:      isDebug,
			})
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&opts.kubeConfig, "kubeconfig", "", "absolute path to the kubernetes config")
	flags.StringVar(&opts.masterURL, "master-url", "", "The address of the Kubernetes API server. Overrides any value in kubeConfig. Only required if out-of-cluster.")
	flags.IntVar(&opts.maxLoadTestsRun, "max-load-tests", 10, "The maximum amount of load tests to run simultaneously.")
	return cmd
}
