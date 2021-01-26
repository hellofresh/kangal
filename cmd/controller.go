package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/spf13/cobra"
	kubeInformers "k8s.io/client-go/informers"
	kubeClient "k8s.io/client-go/kubernetes"

	"github.com/hellofresh/kangal/pkg/controller"
	"github.com/hellofresh/kangal/pkg/core/observability"
	"github.com/hellofresh/kangal/pkg/kubernetes"
	clientSet "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
	informers "github.com/hellofresh/kangal/pkg/kubernetes/generated/informers/externalversions"
)

type controllerCmdOptions struct {
	kubeConfig           string
	masterURL            string
	namespaceAnnotations []string
	podAnnotations       []string
}

// NewControllerCmd creates a new proxy command
func NewControllerCmd() *cobra.Command {
	opts := &controllerCmdOptions{}

	cmd := &cobra.Command{
		Use:     "controller",
		Short:   "Run controller to communicate to k8s infrastructure",
		Aliases: []string{"c"},
		RunE: func(cmd *cobra.Command, args []string) error {
			var cfg controller.Config
			if err := envconfig.Process("", &cfg); err != nil {
				return fmt.Errorf("could not load config from env: %w", err)
			}

			// set some command line option to controller config
			cfg, err := populateCfgFromOpts(cfg, opts)
			if err != nil {
				return err
			}

			logger, _, err := observability.NewLogger(cfg.Logger)
			if err != nil {
				return fmt.Errorf("could not build logger instance: %w", err)
			}

			pe, err := observability.NewPrometheusExporter("kangal-controller", observability.ControllerViews)
			if err != nil {
				return err
			}

			kubeCfg, err := kubernetes.BuildKubeClientConfig(cfg.MasterURL, cfg.KubeConfig, cfg.KubeClientTimeout)
			if err != nil {
				return fmt.Errorf("error building kubeConfig: %w", err)
			}

			kubeClient, err := kubeClient.NewForConfig(kubeCfg)
			if err != nil {
				return fmt.Errorf("error building kubernetes clientSet: %w", err)
			}

			kangalClient, err := clientSet.NewForConfig(kubeCfg)
			if err != nil {
				return fmt.Errorf("error building kangal clientSet: %w", err)
			}

			statsClient, err := observability.NewStatsReporter("kangal")
			if err != nil {
				return fmt.Errorf("error getting stats client:  %w", err)
			}

			kubeInformerFactory := kubeInformers.NewSharedInformerFactory(kubeClient, time.Second*30)
			kangalInformerFactory := informers.NewSharedInformerFactory(kangalClient, time.Second*30)

			return controller.Run(cfg, controller.Runner{
				Logger:         logger,
				Exporter:       pe,
				KubeClient:     kubeClient,
				KangalClient:   kangalClient,
				StatsReporter:  statsClient,
				KubeInformer:   kubeInformerFactory,
				KangalInformer: kangalInformerFactory,
			})
		},
	}

	flags := cmd.PersistentFlags()
	flags.StringVar(&opts.kubeConfig, "kubeconfig", "", "(optional) Absolute path to the kubeConfig file. Only required if out-of-cluster.")
	flags.StringVar(&opts.masterURL, "master-url", "", "The address of the Kubernetes API server. Overrides any value in kubeConfig. Only required if out-of-cluster.")
	flags.StringSliceVar(&opts.namespaceAnnotations, "namespace-annotation", []string{}, "annotation will be attached to the loadtest namespace")
	flags.StringSliceVar(&opts.podAnnotations, "pod-annotation", []string{}, "annotation will be attached to the loadtest pods")

	return cmd
}

func populateCfgFromOpts(cfg controller.Config, opts *controllerCmdOptions) (controller.Config, error) {
	var err error

	cfg.MasterURL = opts.masterURL
	cfg.KubeConfig = opts.kubeConfig

	cfg.NamespaceAnnotations, err = convertAnnotationToMap(opts.namespaceAnnotations)
	if err != nil {
		return controller.Config{}, fmt.Errorf("failed to convert namepsace annotations: %w", err)
	}
	cfg.PodAnnotations, err = convertAnnotationToMap(opts.podAnnotations)
	if err != nil {
		return controller.Config{}, fmt.Errorf("failed to convert pod annotations: %w", err)
	}
	return cfg, nil
}

func convertAnnotationToMap(s []string) (map[string]string, error) {
	m := map[string]string{}
	for _, a := range s {
		// We need to split annotation string to key value map and remove special chars from it:
		// Before string: iam.amazonaws.com/role: "arn:aws:iam::id:role/some-role"
		// After map[string]string: iam.amazonaws.com/role -> arn:aws:iam::id:role/some-role
		a = strings.Replace(a, `"`, ``, -1)
		str := strings.SplitN(a, ":", 2)
		if len(str) < 2 {
			return nil, fmt.Errorf(fmt.Sprintf("Annotation %q is invalid", a))
		}
		key, value := str[0], str[1]
		m[key] = value
	}
	return m, nil
}
