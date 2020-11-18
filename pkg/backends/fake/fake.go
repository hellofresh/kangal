package fake

import (
	"context"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	defaultImageName = "alpine"
	defaultImageTag  = "3.12.0"
)

// Fake enables the controller to run a LoadTest using Fake load provider which simulates load test
type Fake struct {
	kubeClient kubernetes.Interface
	logger     *zap.Logger
	config     loadTestV1.ImageDetails
}

// Type returns backend type name
func (*Fake) Type() loadTestV1.LoadTestType {
	return loadTestV1.LoadTestTypeFake
}

// SetDefaults must set default values
func (b *Fake) SetDefaults() {
	b.config = loadTestV1.ImageDetails{
		Image: defaultImageName,
		Tag:   defaultImageTag,
	}
}

// SetLogger recieves a copy of logger
func (b *Fake) SetLogger(logger *zap.Logger) {
	b.logger = logger
}

// TransformLoadTestSpec use given spec to validate and return a new one or error
func (*Fake) TransformLoadTestSpec(spec *loadTestV1.LoadTestSpec) error {
	spec.MasterConfig.Image = defaultImageName
	spec.MasterConfig.Tag = defaultImageTag

	spec.WorkerConfig.Image = ""
	spec.WorkerConfig.Tag = ""

	return nil
}

// SetKubeClientSet recieves a copy of kubeClientSet
func (b *Fake) SetKubeClientSet(kubeClientSet kubernetes.Interface) {
	b.kubeClient = kubeClientSet
}

// Sync check if Fake kubernetes resources have been create, if they have not been create them
func (b *Fake) Sync(ctx context.Context, loadTest loadTestV1.LoadTest, reportURL string) error {
	// Get the Namespace resource
	namespace, err := b.kubeClient.CoreV1().Namespaces().Get(ctx, loadTest.Status.Namespace, metaV1.GetOptions{})
	// The LoadTest resource may no longer exist, in which case we stop
	// processing.
	if errors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// Check that we created master job
	_, err = b.kubeClient.BatchV1().Jobs(namespace.GetName()).Get(ctx, "loadtest-master", metaV1.GetOptions{})
	if errors.IsNotFound(err) {
		_, err = b.kubeClient.BatchV1().Jobs(namespace.GetName()).Create(ctx, b.NewMasterJob(loadTest), metaV1.CreateOptions{})
		return err
	}
	return err
}

// SyncStatus check the Fake resources and calculate the current status of the LoadTest from them
func (b *Fake) SyncStatus(ctx context.Context, loadTest loadTestV1.LoadTest, loadTestStatus *loadTestV1.LoadTestStatus) error {
	// Get the Namespace resource
	namespace, err := b.kubeClient.CoreV1().Namespaces().Get(ctx, loadTestStatus.Namespace, metaV1.GetOptions{})
	// The LoadTest resource may no longer exist, in which case we stop
	// processing.
	if errors.IsNotFound(err) {
		loadTestStatus.Phase = loadTestV1.LoadTestFinished
		return nil
	}
	if err != nil {
		return err
	}

	if loadTestStatus.Phase == "" {
		loadTestStatus.Phase = loadTestV1.LoadTestCreating
	}

	if loadTestStatus.Phase == loadTestV1.LoadTestErrored {
		return nil
	}

	job, err := b.kubeClient.BatchV1().Jobs(namespace.GetName()).Get(ctx, "loadtest-master", metaV1.GetOptions{})
	if err != nil {
		return err
	}

	// Get Fake job in namespace and update the LoadTest status with
	// the Job status
	loadTestStatus.Phase = getLoadTestPhaseFromJob(job.Status)
	loadTestStatus.JobStatus = job.Status

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
