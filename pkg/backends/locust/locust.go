package locust

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

var (
	defaultImageName = "locustio/locust"
	defaultImageTag  = "latest"
)

// Locust enables the controller to run a loadtest using locust.io
type Locust struct {
	backend   Backend
	loadTest  *loadTestV1.LoadTest
	reportURL string
}

// CheckOrCreateResources check for resources or create the needed resources for the loadtest type
func (c *Locust) CheckOrCreateResources(ctx context.Context) error {
	return c.backend.Sync(ctx, *c.loadTest, c.reportURL)
}

// CheckOrUpdateStatus check current LoadTest progress
func (c *Locust) CheckOrUpdateStatus(ctx context.Context) error {
	return c.backend.SyncStatus(ctx, *c.loadTest, &c.loadTest.Status)
}

// New creates a instance of Locust backend
func New(
	kubeClientSet kubernetes.Interface,
	_ clientSetV.Interface,
	loadTest *loadTestV1.LoadTest,
	logger *zap.Logger,
	reportURL string,
	config Config,
	podAnnotations map[string]string,
) *Locust {
	backend := Backend{
		logger:         logger,
		kubeClientSet:  kubeClientSet,
		config:         &config,
		podAnnotations: podAnnotations,
	}

	return &Locust{
		backend:   backend,
		loadTest:  loadTest,
		reportURL: reportURL,
	}
}
