package locust

import (
	"context"
	"errors"
	"fmt"

	"go.uber.org/zap"
	coreV1 "k8s.io/api/core/v1"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hellofresh/kangal/pkg/backends"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	defaultImageName = "locustio/locust"
	defaultImageTag  = "latest"
)

var (
	// ErrRequireMinOneDistributedPod spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

func init() {
	backends.Register(&Backend{})
}

// Backend is the Locust implementation of backend interface
type Backend struct {
	logger         *zap.Logger
	kubeClientSet  kubernetes.Interface
	config         *Config
	podAnnotations map[string]string

	// defined on SetDefaults
	image           loadTestV1.ImageDetails
	masterResources backends.Resources
	workerResources backends.Resources
}

// Type returns backend type name
func (*Backend) Type() loadTestV1.LoadTestType {
	return loadTestV1.LoadTestTypeLocust
}

// GetEnvConfig must return config struct pointer
func (b *Backend) GetEnvConfig() interface{} {
	b.config = &Config{}
	return b.config
}

// SetDefaults must set default values
func (b *Backend) SetDefaults() {
	// this ensure backward compatibility
	if b.config.ImageName == "" && b.config.Image != "" {
		b.config.ImageName = b.config.Image
	}

	if b.config.ImageName == "" || b.config.ImageTag == "" {
		b.config.ImageName = defaultImageName
		b.config.ImageTag = defaultImageTag
	}

	b.image = loadTestV1.ImageDetails{
		Image: b.config.ImageName,
		Tag:   b.config.ImageTag,
	}

	b.workerResources = backends.Resources{
		CPULimits:      b.config.WorkerCPULimits,
		CPURequests:    b.config.WorkerCPURequests,
		MemoryLimits:   b.config.WorkerMemoryLimits,
		MemoryRequests: b.config.WorkerMemoryRequests,
	}

	b.masterResources = backends.Resources{
		CPULimits:      b.config.MasterCPULimits,
		CPURequests:    b.config.MasterCPURequests,
		MemoryLimits:   b.config.MasterMemoryLimits,
		MemoryRequests: b.config.MasterMemoryRequests,
	}
}

// SetPodAnnotations receives a copy of pod annotations
func (b *Backend) SetPodAnnotations(podAnnotations map[string]string) {
	b.podAnnotations = podAnnotations
}

// SetKubeClientSet receives a copy of kubeClientSet
func (b *Backend) SetKubeClientSet(kubeClientSet kubernetes.Interface) {
	b.kubeClientSet = kubeClientSet
}

// SetLogger receives a copy of logger
func (b *Backend) SetLogger(logger *zap.Logger) {
	b.logger = logger
}

// TransformLoadTestSpec use given spec to validate and return a new one or error
func (b *Backend) TransformLoadTestSpec(spec *loadTestV1.LoadTestSpec) error {
	if nil == spec.DistributedPods {
		return ErrRequireMinOneDistributedPod
	}

	if *spec.DistributedPods <= int32(0) {
		return ErrRequireMinOneDistributedPod
	}

	if spec.TestFile == "" {
		return ErrRequireTestFile
	}

	if spec.MasterConfig.Image == "" || spec.MasterConfig.Tag == "" {
		spec.MasterConfig.Image = b.image.Image
		spec.MasterConfig.Tag = b.image.Tag
	}

	if spec.WorkerConfig.Image == "" || spec.WorkerConfig.Tag == "" {
		spec.WorkerConfig.Image = b.image.Image
		spec.WorkerConfig.Tag = b.image.Tag
	}

	return nil
}

// Sync check if Backend kubernetes resources have been create, if they have not been create them
func (b *Backend) Sync(ctx context.Context, loadTest loadTestV1.LoadTest, reportURL string) error {
	workerJobs, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTest.Status.Namespace).
		List(ctx, metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", newWorkerJobName(loadTest)),
		})
	if err != nil {
		b.logger.Error("Error on listing jobs", zap.Error(err))
		return err
	}

	if len(workerJobs.Items) > 0 {
		return nil
	}

	configMap := newConfigMap(loadTest)
	_, err = b.kubeClientSet.
		CoreV1().
		ConfigMaps(loadTest.Status.Namespace).
		Create(ctx, configMap, metaV1.CreateOptions{})
	if err != nil && !k8sAPIErrors.IsAlreadyExists(err) {
		b.logger.Error("Error on creating testfile configmap", zap.Error(err))
		return err
	}

	var secret *coreV1.Secret

	if loadTest.Spec.EnvVars != nil {
		secret = newSecret(loadTest, loadTest.Spec.EnvVars)
		_, err = b.kubeClientSet.
			CoreV1().
			Secrets(loadTest.Status.Namespace).
			Create(ctx, secret, metaV1.CreateOptions{})
		if err != nil && !k8sAPIErrors.IsAlreadyExists(err) {
			b.logger.Error("Error on creating secret", zap.Error(err))
			return err
		}
	}

	masterJob := newMasterJob(loadTest, configMap, secret, reportURL, b.masterResources, b.podAnnotations, b.image, b.logger)
	_, err = b.kubeClientSet.
		BatchV1().
		Jobs(loadTest.Status.Namespace).
		Create(ctx, masterJob, metaV1.CreateOptions{})
	if err != nil && !k8sAPIErrors.IsAlreadyExists(err) {
		b.logger.Error("Error on creating master job", zap.Error(err))
		return err
	}

	masterService := newMasterService(loadTest, masterJob)
	_, err = b.kubeClientSet.CoreV1().Services(loadTest.Status.Namespace).Create(ctx, masterService, metaV1.CreateOptions{})
	if err != nil && !k8sAPIErrors.IsAlreadyExists(err) {
		b.logger.Error("Error on creating master service", zap.Error(err))
		return err
	}

	workerJob := newWorkerJob(loadTest, configMap, secret, masterService, b.workerResources, b.podAnnotations, b.image, b.logger)
	_, err = b.kubeClientSet.
		BatchV1().
		Jobs(loadTest.Status.Namespace).
		Create(ctx, workerJob, metaV1.CreateOptions{})
	if err != nil && !k8sAPIErrors.IsAlreadyExists(err) {
		b.logger.Error("Error on creating worker job", zap.Error(err))
		return err
	}

	return nil
}

// SyncStatus check the Backend resources and calculate the current status of the LoadTest from them
func (b *Backend) SyncStatus(ctx context.Context, loadTest loadTestV1.LoadTest, loadTestStatus *loadTestV1.LoadTestStatus) error {
	if loadTestStatus.Phase == "" {
		loadTestStatus.Phase = loadTestV1.LoadTestCreating
	}

	if loadTestStatus.Phase == loadTestV1.LoadTestErrored {
		return nil
	}

	_, err := b.kubeClientSet.
		CoreV1().
		ConfigMaps(loadTestStatus.Namespace).
		Get(ctx, newConfigMapName(loadTest), metaV1.GetOptions{})
	if err != nil {
		if k8sAPIErrors.IsNotFound(err) {
			loadTestStatus.Phase = loadTestV1.LoadTestFinished
			return nil
		}
		return err
	}

	workerJob, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTestStatus.Namespace).
		Get(ctx, newWorkerJobName(loadTest), metaV1.GetOptions{})
	if err != nil {
		return err
	}

	masterJob, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTestStatus.Namespace).
		Get(ctx, newMasterJobName(loadTest), metaV1.GetOptions{})
	if err != nil {
		return err
	}

	loadTestStatus.Phase = determineLoadTestStatusFromJobs(masterJob, workerJob)
	loadTestStatus.JobStatus = masterJob.Status

	return nil
}
