package locust

import (
	"context"
	"errors"
	"fmt"

	coreV1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hellofresh/kangal/pkg/core/helper"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"go.uber.org/zap"
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

// Locust enables the controller to run a loadtest using locust.io
type Locust struct {
	logger          *zap.Logger
	kubeClientSet   kubernetes.Interface
	config          loadTestV1.ImageDetails
	masterResources helper.Resources
	workerResources helper.Resources
	podAnnotations  map[string]string
	envConfig       *Config
}

// Type returns backend type name
func (*Locust) Type() loadTestV1.LoadTestType {
	return loadTestV1.LoadTestTypeLocust
}

// GetEnvConfig must return envConfig struct pointer
func (b *Locust) GetEnvConfig() interface{} {
	b.envConfig = &Config{}
	return b.envConfig
}

// SetDefaults must set default values
func (b *Locust) SetDefaults() {
	// this ensure backward compatibility
	if b.envConfig.ImageName == "" && b.envConfig.Image != "" {
		b.envConfig.ImageName = b.envConfig.Image
	}

	if b.envConfig.ImageName == "" || b.envConfig.ImageTag == "" {
		b.envConfig.ImageName = defaultImageName
		b.envConfig.ImageTag = defaultImageTag
	}

	b.config = loadTestV1.ImageDetails{
		Image: b.envConfig.ImageName,
		Tag:   b.envConfig.ImageTag,
	}

	b.workerResources = helper.Resources{
		CPULimits:      b.envConfig.WorkerCPULimits,
		CPURequests:    b.envConfig.WorkerCPURequests,
		MemoryLimits:   b.envConfig.WorkerMemoryLimits,
		MemoryRequests: b.envConfig.WorkerMemoryRequests,
	}

	b.masterResources = helper.Resources{
		CPULimits:      b.envConfig.MasterCPULimits,
		CPURequests:    b.envConfig.MasterCPURequests,
		MemoryLimits:   b.envConfig.MasterMemoryLimits,
		MemoryRequests: b.envConfig.MasterMemoryRequests,
	}
}

// SetLogger recieves a copy of logger
func (b *Locust) SetLogger(logger *zap.Logger) {
	b.logger = logger
}

// TransformLoadTestSpec use given spec to validate and return a new one or error
func (b *Locust) TransformLoadTestSpec(spec *loadTestV1.LoadTestSpec) error {
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
		spec.MasterConfig.Image = b.config.Image
		spec.MasterConfig.Tag = b.config.Tag
	}

	if spec.WorkerConfig.Image == "" || spec.WorkerConfig.Tag == "" {
		spec.WorkerConfig.Image = b.config.Image
		spec.WorkerConfig.Tag = b.config.Tag
	}

	return nil
}

// SetPodAnnotations recieves a copy of pod annotations
func (b *Locust) SetPodAnnotations(podAnnotations map[string]string) {
	b.podAnnotations = podAnnotations
}

// SetKubeClientSet recieves a copy of kubeClientSet
func (b *Locust) SetKubeClientSet(kubeClientSet kubernetes.Interface) {
	b.kubeClientSet = kubeClientSet
}

// Sync check for resources or create the needed resources for the loadtest type
func (b *Locust) Sync(ctx context.Context, loadTest loadTestV1.LoadTest, reportURL string) error {
	workerJobs, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTest.Status.Namespace).
		List(ctx, metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", b.newWorkerJobName(loadTest)),
		})
	if err != nil {
		b.logger.Error("Error on listing jobs", zap.Error(err))
		return err
	}

	if len(workerJobs.Items) > 0 {
		return nil
	}

	configMap := b.newConfigMap(loadTest)
	_, err = b.kubeClientSet.
		CoreV1().
		ConfigMaps(loadTest.Status.Namespace).
		Create(ctx, configMap, metaV1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		b.logger.Error("Error on creating testfile configmap", zap.Error(err))
		return err
	}

	var secret *coreV1.Secret

	if loadTest.Spec.EnvVars != "" {
		envs, err := helper.ReadEnvs(loadTest.Spec.EnvVars)
		if err != nil {
			b.logger.Error("Error reading envVars", zap.Error(err))
			return err
		}
		if len(envs) > 0 {
			secret = b.newSecret(loadTest, envs)
			if nil != err {
				b.logger.Error("Error on creating secret", zap.Error(err))
				return err
			}
			_, err = b.kubeClientSet.
				CoreV1().
				Secrets(loadTest.Status.Namespace).
				Create(ctx, secret, metaV1.CreateOptions{})
			if err != nil && !kerrors.IsAlreadyExists(err) {
				b.logger.Error("Error on creating testfile configmap", zap.Error(err))
				return err
			}
		}
	}

	masterJob := b.newMasterJob(loadTest, configMap, secret, reportURL)
	_, err = b.kubeClientSet.
		BatchV1().
		Jobs(loadTest.Status.Namespace).
		Create(ctx, masterJob, metaV1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		b.logger.Error("Error on creating master job", zap.Error(err))
		return err
	}

	masterService := b.newMasterService(loadTest, masterJob)
	_, err = b.kubeClientSet.CoreV1().Services(loadTest.Status.Namespace).Create(ctx, masterService, metaV1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		b.logger.Error("Error on creating master service", zap.Error(err))
		return err
	}

	workerJob := b.newWorkerJob(loadTest, configMap, secret, masterService)
	_, err = b.kubeClientSet.
		BatchV1().
		Jobs(loadTest.Status.Namespace).
		Create(ctx, workerJob, metaV1.CreateOptions{})
	if err != nil && !kerrors.IsAlreadyExists(err) {
		b.logger.Error("Error on creating worker job", zap.Error(err))
		return err
	}

	return nil
}

// SyncStatus check current LoadTest progress
func (b *Locust) SyncStatus(ctx context.Context, loadTest loadTestV1.LoadTest, loadTestStatus *loadTestV1.LoadTestStatus) error {
	if loadTestStatus.Phase == "" {
		loadTestStatus.Phase = loadTestV1.LoadTestCreating
	}

	if loadTestStatus.Phase == loadTestV1.LoadTestErrored ||
		loadTestStatus.Phase == loadTestV1.LoadTestFinished {
		return nil
	}

	_, err := b.kubeClientSet.
		CoreV1().
		ConfigMaps(loadTestStatus.Namespace).
		Get(ctx, b.newConfigMapName(loadTest), metaV1.GetOptions{})
	if err != nil {
		if kerrors.IsNotFound(err) {
			loadTestStatus.Phase = loadTestV1.LoadTestFinished
			return nil
		}
		return err
	}

	workerJob, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTestStatus.Namespace).
		Get(ctx, b.newWorkerJobName(loadTest), metaV1.GetOptions{})
	if err != nil {
		return err
	}

	masterJob, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTestStatus.Namespace).
		Get(ctx, b.newMasterJobName(loadTest), metaV1.GetOptions{})
	if err != nil {
		return err
	}

	loadTestStatus.Phase = b.getLoadTestStatusFromJobs(masterJob, workerJob)

	return nil
}
