package fake

import (
	"context"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	"k8s.io/client-go/kubernetes"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	imageName = "alpine"
	imageTag  = "3.12.0"
)

// Fake enables the controller to run a LoadTest using Fake load provider which simulates load test
type Fake struct {
	backend    *Backend
	kubeClient kubernetes.Interface
	loadTest   *loadTestV1.LoadTest
	logger     *zap.Logger
}

//New initializes new Fake provider handler to manage load test resources with Kangal Controller
func New(kubeClientSet kubernetes.Interface, lt *loadTestV1.LoadTest, logger *zap.Logger) *Fake {
	backend := &Backend{
		kubeClient: kubeClientSet,
		logger:     logger,
	}

	backend.SetDefaults()

	return &Fake{
		backend:    backend,
		kubeClient: kubeClientSet,
		loadTest:   lt,
		logger:     logger,
	}
}

// CheckOrCreateResources check if Fake kubernetes resources have been create, if they have not been create them
func (c *Fake) CheckOrCreateResources(ctx context.Context) error {
	return c.backend.Sync(ctx, *c.loadTest, "")
}

// CheckOrUpdateStatus check the Fake resources and calculate the current status of the LoadTest from them
func (c *Fake) CheckOrUpdateStatus(ctx context.Context) error {
	return c.backend.SyncStatus(ctx, *c.loadTest, &c.loadTest.Status)
}

func getLoadTestPhaseFromJob(status batchV1.JobStatus) loadTestV1.LoadTestPhase {
	if status.Active > 0 {
		return loadTestV1.LoadTestRunning
	}

	if status.Succeeded == 0 && status.Failed == 0 {
		return loadTestV1.LoadTestStarting
	}

	return loadTestV1.LoadTestFinished
}
