package k6

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
	// ErrRequireMinOneDistributedPod Backend spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

const (
	defaultImageName = "loadimpact/k6"
	defaultImageTag  = "latest"
)

func init() {
	backends.Register(&Backend{})
}

// Backend is the k6 implementation of backend interface
type Backend struct {
	logger         *zap.Logger
	kubeClientSet  kubernetes.Interface
	config         *Config
	podAnnotations map[string]string
	podTolerations []coreV1.Toleration

	nodeSelector map[string]string
	// defined on SetDefaults
	image     loadTestV1.ImageDetails
	resources backends.Resources
}

// Type returns backend type name
func (*Backend) Type() loadTestV1.LoadTestType {
	return loadTestV1.LoadTestTypeK6
}

func (*Backend) UsesCSVTestData() bool {
	return false
}

// GetEnvConfig must return config struct pointer
func (b *Backend) GetEnvConfig() interface{} {
	b.config = &Config{}
	return b.config
}

// SetDefaults must set default values
func (b *Backend) SetDefaults() {
	if b.config.ImageName == "" || b.config.ImageTag == "" {
		b.config.ImageName = defaultImageName
		b.config.ImageTag = defaultImageTag
	}
	b.image = loadTestV1.ImageDetails{
		Image: b.config.ImageName,
		Tag:   b.config.ImageTag,
	}

	b.resources = backends.Resources{
		CPULimits:      b.config.CPULimits,
		CPURequests:    b.config.CPURequests,
		MemoryLimits:   b.config.MemoryLimits,
		MemoryRequests: b.config.MemoryRequests,
	}
}

// SetPodAnnotations receives a copy of pod annotations
func (b *Backend) SetPodAnnotations(podAnnotations map[string]string) {
	b.podAnnotations = podAnnotations
}

// SetPodTolerations receives a copy of pod tolerations
func (b *Backend) SetPodTolerations(tolerations []coreV1.Toleration) {
	b.podTolerations = tolerations
}

// SetKubeClientSet receives a copy of kubeClientSet
func (b *Backend) SetKubeClientSet(kubeClientSet kubernetes.Interface) {
	b.kubeClientSet = kubeClientSet
}

// SetLogger receives a copy of logger
func (b *Backend) SetLogger(logger *zap.Logger) {
	b.logger = logger
}

// SetPodNodeSelector receives a copy of pod node selectors
func (b *Backend) SetPodNodeSelector(nodeselector map[string]string) {
	b.nodeSelector = nodeselector
}

// TransformLoadTestSpec use given spec to validate and return a new one or error
func (b *Backend) TransformLoadTestSpec(spec *loadTestV1.LoadTestSpec) error {
	if nil == spec.DistributedPods {
		return ErrRequireMinOneDistributedPod
	}

	if *spec.DistributedPods <= int32(0) {
		return ErrRequireMinOneDistributedPod
	}

	if len(spec.TestFile) == 0 {
		return ErrRequireTestFile
	}

	if spec.MasterConfig.Image == "" || spec.MasterConfig.Tag == "" {
		spec.MasterConfig.Image = b.image.Image
		spec.MasterConfig.Tag = b.image.Tag
	}

	return nil
}

// Sync checks if k6 kubernetes resources have been created, create them if they haven't
func (b *Backend) Sync(ctx context.Context, loadTest loadTestV1.LoadTest, testfileConfigMapName string, testdataConfigMapNames []string, reportURL string) error {
	jobs, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTest.Status.Namespace).
		List(ctx, metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", loadTestJobName),
		})
	if err != nil {
		b.logger.Error("Error on listing jobs", zap.Error(err))
		return err
	}

	// Jobs already created, do nothing
	if len(jobs.Items) > 0 {
		return nil
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

	for i := int32(0); i < *loadTest.Spec.DistributedPods; i++ {
		// Prepare Volume and VolumeMount for job creation
		var (
			volumes = make([]coreV1.Volume, 1)
			mounts  = make([]coreV1.VolumeMount, 1)
		)

		volumes[0], mounts[0] = NewFileVolumeAndMount(loadTestFileVolumeName, testfileConfigMapName, backends.LoadTestScript, scriptTestFileName)

		if len(testdataConfigMapNames) > 0 {
			v, m := NewFileVolumeAndMount(loadTestDataVolumeName, testdataConfigMapNames[i%int32(len(testdataConfigMapNames))], backends.LoadTestData, testdataFileName)
			volumes = append(volumes, v)
			mounts = append(mounts, m)
		}
		// Create Job
		job := b.NewJob(loadTest, volumes, mounts, secret, reportURL, i)
		_, err = b.kubeClientSet.
			BatchV1().
			Jobs(loadTest.Status.Namespace).
			Create(ctx, job, metaV1.CreateOptions{})
		if err != nil && !k8sAPIErrors.IsAlreadyExists(err) {
			b.logger.Error("Error on creating master job", zap.Error(err))
			return err
		}
	}

	return nil
}

// SyncStatus checks k6 resources and updates the status of the LoadTest resource
func (b *Backend) SyncStatus(ctx context.Context, _ loadTestV1.LoadTest, loadTestStatus *loadTestV1.LoadTestStatus) error {
	if loadTestStatus.Phase == "" {
		loadTestStatus.Phase = loadTestV1.LoadTestCreating
		return nil
	}

	if loadTestStatus.Phase == loadTestV1.LoadTestErrored {
		return nil
	}

	jobs, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTestStatus.Namespace).
		List(ctx, metaV1.ListOptions{})
	if err != nil {
		return err
	}

	loadTestStatus.Phase = determineLoadTestPhaseFromJobs(jobs.Items)
	loadTestStatus.JobStatus = determineLoadTestStatusFromJobs(jobs.Items)
	return nil
}
