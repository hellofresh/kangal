package fake

import (
	"context"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	coreListersV1 "k8s.io/client-go/listers/core/v1"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
	"github.com/hellofresh/kangal/pkg/report"
)

var (
	sleepImage = "alpine"
	imageTag   = "latest"
)

// Fake enables the controller to run a LoadTest using Fake load provider which simulates load test
type Fake struct {
	kubeClientSet    kubernetes.Interface
	kangalClientSet  clientSetV.Interface
	loadTest         *loadTestV1.LoadTest
	logger           *zap.Logger
	namespacesLister coreListersV1.NamespaceLister
	reportConfig     report.Config

	podAnnotations, namespaceAnnotations map[string]string
}

//New initializes new Fake provider handler to manage load test resources with Kangal Controller
func New(kubeClientSet kubernetes.Interface, kangalClientSet clientSetV.Interface, lt *loadTestV1.LoadTest, logger *zap.Logger, namespacesLister coreListersV1.NamespaceLister, reportConfig report.Config, podAnnotations, namespaceAnnotations map[string]string) *Fake {
	return &Fake{
		kubeClientSet:        kubeClientSet,
		kangalClientSet:      kangalClientSet,
		loadTest:             lt,
		logger:               logger,
		namespacesLister:     namespacesLister,
		reportConfig:         reportConfig,
		podAnnotations:       podAnnotations,
		namespaceAnnotations: namespaceAnnotations,
	}
}

// SetDefaults set default values for creating a Fake LoadTest pods
func (c *Fake) SetDefaults() error {
	if c.loadTest.Status.Phase == "" {
		c.loadTest.Status.Phase = loadTestV1.LoadTestCreating
	}

	if c.loadTest.Spec.MasterConfig.Image == "" {
		c.loadTest.Spec.MasterConfig.Image = sleepImage
	}

	if c.loadTest.Spec.MasterConfig.Tag == "" {
		c.loadTest.Spec.MasterConfig.Tag = imageTag
	}

	return nil
}

// CheckOrCreateResources check if Fake kubernetes resources have been create, if they have not been create them
func (c *Fake) CheckOrCreateResources(ctx context.Context) error {
	return nil
}

// CheckOrUpdateStatus check the Fake resources and calculate the current status of the LoadTest from them
func (c *Fake) CheckOrUpdateStatus(ctx context.Context) error {
	// Get the Namespace resource
	namespace, err := c.namespacesLister.Get(c.loadTest.Status.Namespace)
	if err != nil {
		// The LoadTest resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			c.loadTest.Status.Phase = loadTestV1.LoadTestFinished
			return nil
		}
	}

	if c.loadTest.Status.Phase == loadTestV1.LoadTestErrored {
		return nil
	}

	job, err := c.kubeClientSet.BatchV1().Jobs(namespace.GetName()).Get(ctx, "loadtest-master", metaV1.GetOptions{})
	if err != nil {
		// The LoadTest resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			_, err = c.kubeClientSet.BatchV1().Jobs(namespace.GetName()).Create(
				ctx,
				c.NewMasterJob(
					c.podAnnotations,
				),
				metaV1.CreateOptions{},
			)
			return err
		}
		return err
	}

	// Get Fake job in namespace and update the LoadTest status with
	// the Job status
	c.loadTest.Status.Phase = getLoadTestPhaseFromJob(job.Status)
	c.loadTest.Status.JobStatus = job.Status

	return nil
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
