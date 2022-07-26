package fake

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hellofresh/kangal/pkg/backends"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	imageName = "alpine"
	imageTag  = "3.12.0"
)

func init() {
	backends.Register(&Backend{})
}

// Backend is the Fake implementation of backend interface
type Backend struct {
	kubeClient kubernetes.Interface
	logger     *zap.Logger
	config     loadTestV1.ImageDetails
}

// Type returns backend type name
func (*Backend) Type() loadTestV1.LoadTestType {
	return loadTestV1.LoadTestTypeFake
}

// SetDefaults must set default values
func (b *Backend) SetDefaults() {
	b.config = loadTestV1.ImageDetails(fmt.Sprintf("%s:%s", imageName, imageTag))
}

// SetLogger receives a copy of logger
func (b *Backend) SetLogger(logger *zap.Logger) {
	b.logger = logger
}

// TransformLoadTestSpec use given spec to validate and return a new one or error
func (*Backend) TransformLoadTestSpec(spec *loadTestV1.LoadTestSpec) error {
	spec.MasterConfig = loadTestV1.ImageDetails(fmt.Sprintf("%s:%s", imageName, imageTag))
	spec.WorkerConfig = ""

	return nil
}

// SetKubeClientSet receives a copy of kubeClientSet
func (b *Backend) SetKubeClientSet(kubeClientSet kubernetes.Interface) {
	b.kubeClient = kubeClientSet
}

// Sync check if Fake kubernetes resources have been create, if they have not been create them
func (b *Backend) Sync(ctx context.Context, loadTest loadTestV1.LoadTest, _ string) error {
	// Get the Namespace resource
	namespace, err := b.kubeClient.CoreV1().Namespaces().Get(ctx, loadTest.Status.Namespace, metaV1.GetOptions{})
	// The LoadTest resource may no longer exist, in which case we stop
	// processing.
	if k8sAPIErrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}
	// Check that we created master job
	_, err = b.kubeClient.BatchV1().Jobs(namespace.GetName()).Get(ctx, "loadtest-master", metaV1.GetOptions{})
	if k8sAPIErrors.IsNotFound(err) {
		_, err = b.kubeClient.BatchV1().Jobs(namespace.GetName()).Create(ctx, b.newMasterJob(loadTest), metaV1.CreateOptions{})
		return err
	}
	return err
}

// SyncStatus check the Fake resources and calculate the current status of the LoadTest from them
func (b *Backend) SyncStatus(ctx context.Context, _ loadTestV1.LoadTest, loadTestStatus *loadTestV1.LoadTestStatus) error {
	// Get the Namespace resource
	namespace, err := b.kubeClient.CoreV1().Namespaces().Get(ctx, loadTestStatus.Namespace, metaV1.GetOptions{})
	// The LoadTest resource may no longer exist, in which case we stop
	// processing.
	if k8sAPIErrors.IsNotFound(err) {
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
	loadTestStatus.Phase = determineLoadTestPhaseFromJob(job.Status)
	loadTestStatus.JobStatus = job.Status

	return nil
}

// newMasterJob creates a new job which runs the Fake master pod
func (b *Backend) newMasterJob(loadTest loadTestV1.LoadTest) *batchV1.Job {
	imageRef := loadTest.Spec.MasterConfig
	if loadTest.Spec.MasterConfig == "" {
		imageRef = b.config
		b.logger.Warn("Loadtest.Spec.MasterConfig is empty; using default master image", zap.String("imageRef", string(imageRef)))
	}

	// For fake provider we don't really create load test and just use alpine image with some sleep
	// to simulate load test job. Please don't use Fake provider in production.
	return &batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "loadtest-master",
			Labels: map[string]string{
				"app": "loadtest-master",
			},
			OwnerReferences: []metaV1.OwnerReference{
				*metaV1.NewControllerRef(&loadTest, loadTestV1.SchemeGroupVersion.WithKind("LoadTest")),
			},
		},
		Spec: batchV1.JobSpec{
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"app": "loadtest-master",
					},
				},
				Spec: coreV1.PodSpec{
					RestartPolicy: "Never",
					Containers: []coreV1.Container{
						{
							Name:            "loadtest-master",
							Image:           string(imageRef),
							ImagePullPolicy: "Always",
							Command:         []string{"/bin/sh", "-c", "--"},
							Args:            []string{"sleep 10"},
						},
					},
				},
			},
		},
	}
}

// determineLoadTestPhaseFromJob reads existing job statuses and determines what the loadtest phase should be
func determineLoadTestPhaseFromJob(status batchV1.JobStatus) loadTestV1.LoadTestPhase {
	if status.Active > 0 {
		return loadTestV1.LoadTestRunning
	}

	if status.Succeeded == 0 && status.Failed == 0 {
		return loadTestV1.LoadTestStarting
	}

	return loadTestV1.LoadTestFinished
}
